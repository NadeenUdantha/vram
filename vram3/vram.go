package main

import (
	"fmt"
	"math/rand"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func log(x ...any) {
	//fmt.Println(x...)
}

func logf(f string, x ...any) {
	//fmt.Printf(f, x)
}

type vram struct {
	pid     uint32
	ph      uintptr
	vmrange *addr_range
	ns      map[uint64]string
}

func new_vram(pid uint32, ph uintptr) *vram {
	vram := &vram{
		pid:     pid,
		ph:      ph,
		vmrange: new_addr_range(),
		ns:      make(map[uint64]string),
	}
	return vram
}

type region struct {
	vm    bool
	addr  uint64
	size  uint64
	state PageState
}

func (r *region) String() string {
	return fmt.Sprintf("region[addr=%x,size=%d,state=%v]", r.addr, r.size, r.state)
}

const AllocationGranularity = 64 * 1024
const PageSize = 4 * 1024

func roundUp(x uint64, v uint64) uint64 {
	if x%v == 0 {
		return x
	}
	return x + (v - (x % v))
}

func roundDown(x uint64, v uint64) uint64 {
	if x%v == 0 {
		return x
	}
	return x - (x % v)
}

func (vr *vram) reserve(r *region) error {
	log("reserve", r)
	assert(r.state == PS_MEM_FREE)
	k, _, err := VirtualAllocEx(vr.ph, uintptr(r.addr), uintptr(r.size), uintptr(AT_MEM_RESERVE), windows.PAGE_NOACCESS)
	if k == 0 {
		return err
	}
	r.state = PS_MEM_RESERVE
	return nil
}

func (vr *vram) commit(r *region) error {
	log("commit", r)
	assert(r.state == PS_MEM_RESERVE)
	k, _, err := VirtualAllocEx(vr.ph, uintptr(r.addr), uintptr(r.size), uintptr(AT_MEM_COMMIT), windows.PAGE_NOACCESS)
	if k == 0 {
		return err
	}
	r.state = PS_MEM_COMMIT
	if r.vm {
		n, k := vr.ns[r.addr]
		assert(k)
		throw(windows.CloseHandle(vr.createFile(n + ".open")))
	}
	return nil
}

func (vr *vram) release(r *region) error {
	log("release", r)
	if r.vm {
		assert(r.state == PS_MEM_COMMIT)
		n, k := vr.ns[r.addr]
		assert(k)
		throw(windows.CloseHandle(vr.createFile(n + ".close")))
		delete(vr.ns, r.addr)
		vr.unMapFile(r.addr)
		vr.vmrange.remove(r.addr, r.size)
	} else {
		k, _, err := VirtualFreeEx(vr.ph, uintptr(r.addr), 0, uintptr(FT_MEM_RELEASE))
		if k == 0 {
			return err
		}
	}
	r.state = PS_MEM_FREE
	return nil
}

func (vr *vram) reserve_free_region(sz uint64) *region {
	name, addr := vr.mapFileReserve(sz)
	vr.ns[addr] = name
	r := &region{
		vm:    true,
		addr:  addr,
		size:  sz,
		state: PS_MEM_RESERVE,
	}
	vr.vmrange.add(addr, sz)
	log("reserve_free_region", r)
	return r
}

func (vr *vram) protect(r *region, p Protection) (Protection, error) {
	log("protect", r, p)
	assert(r.state == PS_MEM_COMMIT)
	if r.vm {
		assert(p == windows.PAGE_READONLY || p == windows.PAGE_READWRITE || p == windows.PAGE_EXECUTE_READ || p == windows.PAGE_EXECUTE_READWRITE)
	}
	var oldp Protection
	k, _, err := VirtualProtectEx(vr.ph, uintptr(r.addr), uintptr(r.size), uintptr(p), uintptr(unsafe.Pointer(&oldp)))
	if k == 0 {
		return oldp, err
	}
	return oldp, nil
}

func (vr *vram) query(addr uint64, mbi *MEMORY_BASIC_INFORMATION64) error {
	vm := vr.vmrange.in(addr, 1)
	log("query", addr, vm)
	k, _, err := VirtualQueryEx(vr.ph, uintptr(addr), uintptr(unsafe.Pointer(mbi)), MEMORY_BASIC_INFORMATION64_sz)
	if !vm {
		return err
	}
	assert(k == MEMORY_BASIC_INFORMATION64_sz)
	switch mbi.State {
	case PS_MEM_RESERVE:
		assert(mbi.Type == PT_MEM_PRIVATE)
	case PS_MEM_COMMIT:
		assert(mbi.Type == PT_MEM_MAPPED)
		mbi.Type = PT_MEM_PRIVATE
	default:
		fmt.Printf("%v\n", mbi)
		panic("?")
	}
	return nil
}

func (vr *vram) get_region(addr, sz uint64) *region {
	vm := vr.vmrange.in(addr, sz)
	log("get_region", addr, vm, sz)
	var mbi MEMORY_BASIC_INFORMATION64
	k, _, _ := VirtualQueryEx(vr.ph, uintptr(addr), uintptr(unsafe.Pointer(&mbi)), MEMORY_BASIC_INFORMATION64_sz)
	assert(k == MEMORY_BASIC_INFORMATION64_sz)
	assert(mbi.BaseAddress <= addr && addr+sz <= mbi.BaseAddress+mbi.RegionSize)
	if vm {
		if mbi.State == PS_MEM_COMMIT {
			assert(mbi.Type == PT_MEM_MAPPED)
			return &region{
				vm:    true,
				addr:  addr,
				size:  sz,
				state: PS_MEM_COMMIT,
			}
		}
	} else {
		//if mbi.State == PS_MEM_FREE || mbi.State == PS_MEM_COMMIT
		return &region{
			vm:    false,
			addr:  addr,
			size:  sz,
			state: mbi.State,
		}
	}
	fmt.Printf("%v\n", mbi)
	panic("?")
}

func (vr *vram) createFile(name string) windows.Handle {
	f, err := windows.CreateFile(
		syscall.StringToUTF16Ptr(name),
		syscall.GENERIC_READ|syscall.GENERIC_WRITE|syscall.GENERIC_EXECUTE,
		0, nil,
		syscall.CREATE_ALWAYS,
		windows.FILE_ATTRIBUTE_NORMAL,
		//windows.FILE_ATTRIBUTE_TEMPORARY|windows.FILE_FLAG_DELETE_ON_CLOSE|windows.FILE_FLAG_NO_BUFFERING|windows.FILE_FLAG_WRITE_THROUGH,
		0,
	)
	throw(err)
	return f
}

func (vr *vram) mapFileReserve(sz uint64) (string, uint64) {
	name := fmt.Sprintf("Z:\\%d-%d-%d.vram", vr.pid, sz, rand.Uint64())
	f := vr.createFile(name)
	defer windows.CloseHandle(f)
	//set_sparse(syscall.Handle(f), sz)
	fm, err := windows.CreateFileMapping(f, nil, syscall.PAGE_EXECUTE_READWRITE, uint32(sz>>32), uint32(sz), nil)
	throw(err)
	defer windows.CloseHandle(fm)
	addr2, _, err := MapViewOfFileNuma2(uintptr(fm), vr.ph, 0, 0, uintptr(sz), uintptr(AT_MEM_RESERVE), syscall.PAGE_EXECUTE_READWRITE, NUMA_NO_PREFERRED_NODE)
	if addr2 == 0 {
		throw(err)
	}
	return name, uint64(addr2)
}

func (vr *vram) unMapFile(raddr uint64) {
	kk, _, err := UnmapViewOfFile2(vr.ph, uintptr(raddr), 0)
	if kk == 0 {
		throw(err)
	}
}

func errno(err error) uint32 {
	if err == nil {
		return 0
	}
	return uint32(err.(syscall.Errno))
}

func (vr *vram) VirtualAlloc(addr uint64, size uint64, atype AllocType, protect Protection) (uint64, uint32) {
	var rdm uint64
	if atype&AT_MEM_RESERVE == AT_MEM_RESERVE {
		rdm = AllocationGranularity
	} else {
		rdm = PageSize
	}
	addr = roundDown(addr, rdm)
	size = roundUp(size, PageSize)
	var r *region
	if addr == 0 {
		r = vr.reserve_free_region(size)
	} else {
		r = vr.get_region(addr, size)
	}
	var err error
	if addr != 0 && atype&AT_MEM_RESERVE == AT_MEM_RESERVE {
		err = vr.reserve(r)
	}
	if err == nil && atype&AT_MEM_COMMIT == AT_MEM_COMMIT {
		err = vr.commit(r)
		vr.protect(r, protect)
	}
	logf("alloc(addr=%x,sz=%x,type=%v,protect=%v)=%x %v\n", addr, size, atype, protect, r.addr, err)
	return r.addr, errno(err)
}

func (vr *vram) VirtualProtect(addr uint64, size uint64, protect Protection) (Protection, uint32) {
	addr = roundDown(addr, PageSize)
	size = roundUp(size, PageSize)
	r := vr.get_region(addr, size)
	oldp, err := vr.protect(r, protect)
	logf("protect(addr=%x,sz=%x,protect=%v,oldp=%v)=%v\n", addr, size, protect, oldp, err)
	return oldp, errno(err)
}

func (vr *vram) VirtualFree(addr uint64, size uint64, ftype FreeType) uint32 {
	assert(ftype == FT_MEM_RELEASE && size == 0 && addr%PageSize == 0)
	r := vr.get_region(addr, size)
	err := vr.release(r)
	logf("free(addr=%x,sz=%x,type=%v)=%v\n", r.addr, size, ftype, 1)
	return errno(err)
}

func (vr *vram) VirtualQuery(addr uint64, mbi *MEMORY_BASIC_INFORMATION64, insz uint64) uint32 {
	addr = roundDown(addr, PageSize)
	err := vr.query(addr, mbi)
	logf("query(addr=%x)=(base=%x,abase=%x,sz=%x,state=%v,protect=%v,aprotect=%v,type=%v)\n", addr, mbi.BaseAddress, mbi.AllocationBase, mbi.RegionSize, mbi.State, mbi.Protect, mbi.AllocationProtect, mbi.Type)
	return errno(err)
}

func (vr *vram) close() {
	logf("[vram] close\n")
	syscall.CloseHandle(syscall.Handle(vr.ph))
	windows.TerminateProcess(windows.Handle(vr.ph), 0)
}

var kernelbase = syscall.NewLazyDLL("kernelbase.dll")
var VirtualAllocEx = kernel32.NewProc("VirtualAllocEx").Call
var VirtualProtectEx = kernel32.NewProc("VirtualProtectEx").Call
var VirtualFreeEx = kernel32.NewProc("VirtualFreeEx").Call
var VirtualQuery = kernel32.NewProc("VirtualQuery").Call
var VirtualQueryEx = kernel32.NewProc("VirtualQueryEx").Call
var MapViewOfFileNuma2 = kernelbase.NewProc("MapViewOfFileNuma2").Call
var UnmapViewOfFile2 = kernelbase.NewProc("UnmapViewOfFile2").Call

var SetFilePointerEx = kernel32.NewProc("SetFilePointerEx")

func set_sparse(fd syscall.Handle, sz uint64) {
	const FSCTL_SET_SPARSE = 590020
	const FSCTL_SET_ZERO_DATA = 622792
	var bytesReturned uint32
	err := syscall.DeviceIoControl(fd,
		FSCTL_SET_SPARSE,
		nil,
		0,
		nil,
		0,
		&bytesReturned,
		nil)
	throw(err)
	x := make([]uint64, 2)
	x[0] = 0
	x[1] = uint64(sz)
	err = syscall.DeviceIoControl(fd,
		FSCTL_SET_ZERO_DATA,
		(*byte)(unsafe.Pointer(&x[0])),
		16,
		nil,
		0,
		&bytesReturned,
		nil)
	throw(err)
	k, _, err := SetFilePointerEx.Call(uintptr(fd), uintptr(sz), 0, 0)
	if k == 0 {
		throw(err)
	}
	throw(syscall.SetEndOfFile(fd))
}

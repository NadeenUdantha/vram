package main

import (
	"bytes"
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var kernel32 = windows.NewLazySystemDLL("kernel32.dll")
var GetNamedPipeClientProcessId = kernel32.NewProc("GetNamedPipeClientProcessId").Call
var DisconnectNamedPipe = kernel32.NewProc("DisconnectNamedPipe").Call

type vram_pipe_msg struct {
	Retval  uint64
	Addr    uint64
	Size    uint64
	Type    uint32
	Protect Protection
	Err     uint32
	method  [32]byte
}

func handle_pipe(pipe windows.Handle, pid uint32) {
	/*defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			debug.PrintStack()
		}
	}()*/
	defer windows.CloseHandle(pipe)
	defer DisconnectNamedPipe(uintptr(pipe))
	vr := vs.create(pid, readHandle(syscall.Handle(pipe)))
	defer vr.close()
	defer vs.remove(pid)
	fmt.Println("[pipe] connected", pid)
	var m vram_pipe_msg
	msgsz := int(unsafe.Sizeof(m))
	msgb := unsafe.Slice((*byte)(unsafe.Pointer(&m)), msgsz)
	for {
		var n uint32
		err := syscall.ReadFile(syscall.Handle(pipe), msgb, &n, nil)
		if err != nil {
			fmt.Println(err)
			break
		}
		throw(err)
		assert(int(n) == msgsz)
		f := string(m.method[:bytes.IndexByte(m.method[:], 0)])
		//fmt.Println("[pipe] read", pid, pretty.Sprint(m), f)
		switch f {
		case "VirtualAlloc":
			m.Retval, m.Err = vr.VirtualAlloc(m.Addr, m.Size, AllocType(m.Type), m.Protect)
		case "VirtualProtect":
			m.Protect, m.Err = vr.VirtualProtect(m.Addr, m.Size, m.Protect)
			if m.Err == 0 {
				m.Retval = 1
			} else {
				m.Retval = 0
			}
		case "VirtualFree":
			m.Err = vr.VirtualFree(m.Addr, m.Size, FreeType(m.Type))
			if m.Err == 0 {
				m.Retval = 1
			} else {
				m.Retval = 0
			}
		case "VirtualQuery":
			var x MEMORY_BASIC_INFORMATION64
			z := m.Retval
			m.Err = vr.VirtualQuery(m.Addr, &x, MEMORY_BASIC_INFORMATION64_sz)
			if m.Err == 0 {
				m.Retval = MEMORY_BASIC_INFORMATION64_sz
				var w uintptr
				throw(windows.WriteProcessMemory(windows.Handle(vr.ph), uintptr(z), (*byte)(unsafe.Pointer(&x)), MEMORY_BASIC_INFORMATION64_sz, &w))
				assert(w == MEMORY_BASIC_INFORMATION64_sz)
			} else {
				m.Retval = 0
			}
		default:
			panic(f)
		}
		//fmt.Println("[pipe] write", pid, pretty.Sprint(m), f)
		err = syscall.WriteFile(syscall.Handle(pipe), msgb, &n, nil)
		if err != nil {
			fmt.Println(err)
			break
		}
		assert(int(n) == msgsz)
	}
}

func readHandle(pipe syscall.Handle) uintptr {
	var m vram_pipe_msg
	msgsz := int(unsafe.Sizeof(m))
	msgb := unsafe.Slice((*byte)(unsafe.Pointer(&m)), msgsz)
	var n uint32
	throw(syscall.ReadFile(pipe, msgb, &n, nil))
	assert(n == uint32(msgsz))
	throw(syscall.WriteFile(pipe, msgb, &n, nil))
	assert(n == uint32(msgsz))
	return uintptr(m.Retval)
}

const ERROR_PIPE_CONNECTED = 535

func init() {
	go pipes()
}

func pipes() {
	fmt.Println("pipes")
	for {
		pipe, err := windows.CreateNamedPipe(
			windows.StringToUTF16Ptr("\\\\.\\pipe\\vramKNUS"),
			windows.PIPE_ACCESS_DUPLEX,
			windows.PIPE_TYPE_MESSAGE,
			1, //windows.PIPE_UNLIMITED_INSTANCES,
			0,
			0,
			0,
			nil,
		)
		throw(err)
		err = windows.ConnectNamedPipe(pipe, nil)
		if errors.Is(err, syscall.Errno(ERROR_PIPE_CONNECTED)) {
			throw(err)
		}
		var pid uint64
		r1, _, err := GetNamedPipeClientProcessId(uintptr(pipe), uintptr(unsafe.Pointer(&pid)))
		if r1 == 0 {
			throw(err)
		}
		handle_pipe(pipe, uint32(pid))
	}
}

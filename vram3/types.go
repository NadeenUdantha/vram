package main

import (
	"fmt"

	"golang.org/x/sys/windows"
)

type MEMORY_BASIC_INFORMATION64 struct {
	BaseAddress       uint64
	AllocationBase    uint64
	AllocationProtect Protection
	_                 uint32
	RegionSize        uint64
	State             PageState
	Protect           Protection
	Type              PageType
	_                 uint32
}

const MEMORY_BASIC_INFORMATION64_sz = 48

const NUMA_NO_PREFERRED_NODE = 0xffffffff

type AllocType uint32

const (
	AT_MEM_RESERVE  AllocType = windows.MEM_RESERVE
	AT_MEM_COMMIT   AllocType = windows.MEM_COMMIT
	AT_MEM_TOP_DOWN AllocType = windows.MEM_TOP_DOWN
)

func (x AllocType) String() string {
	switch x {
	case AT_MEM_RESERVE:
		return "RESERVE"
	case AT_MEM_COMMIT:
		return "COMMIT"
	case AT_MEM_RESERVE | AT_MEM_COMMIT:
		return "RESERVE+COMMIT"
	case AT_MEM_RESERVE | AT_MEM_TOP_DOWN:
		return "RESERVE+TOP_DOWN"
	default:
		fmt.Println(uintptr(x))
		panic(x)
	}
}

type FreeType uint32

const (
	FT_MEM_RELEASE FreeType = windows.MEM_RELEASE
)

func (x FreeType) String() string {
	switch x {
	case FT_MEM_RELEASE:
		return "RELEASE"
	default:
		fmt.Println(uintptr(x))
		panic(x)
	}
}

type Protection uint32

func (x Protection) String() string {
	switch x {
	case 0:
		return "0"
	case windows.PAGE_NOACCESS:
		return "NOACCESS"
	case windows.PAGE_READONLY:
		return "READ"
	case windows.PAGE_READWRITE:
		return "READ+WRITE"
	case windows.PAGE_EXECUTE_READ:
		return "EXEC+READ"
	case windows.PAGE_EXECUTE_WRITECOPY:
		return "EXEC+WRITECOPY"
	case windows.PAGE_EXECUTE_READWRITE:
		return "EXEC+READ+WRITE"
	default:
		fmt.Println(uintptr(x))
		panic(x)
	}
}

type PageState uint32

const (
	PS_MEM_FREE    PageState = 0x10000
	PS_MEM_RESERVE PageState = windows.MEM_RESERVE
	PS_MEM_COMMIT  PageState = windows.MEM_COMMIT
)

func (x PageState) String() string {
	switch x {
	case PS_MEM_FREE:
		return "FREE"
	case PS_MEM_RESERVE:
		return "RESERVE"
	case PS_MEM_COMMIT:
		return "COMMIT"
	default:
		fmt.Println(uintptr(x))
		panic(x)
	}
}

type PageType uint32

const (
	PT_MEM_IMAGE   PageType = 0x1000000
	PT_MEM_MAPPED  PageType = 0x40000
	PT_MEM_PRIVATE PageType = 0x20000
)

func (x PageType) String() string {
	switch x {
	case 0:
		return "0"
	case PT_MEM_IMAGE:
		return "IMAGE"
	case PT_MEM_MAPPED:
		return "MAPPED"
	case PT_MEM_PRIVATE:
		return "PRIVATE"
	default:
		fmt.Println(uintptr(x))
		panic(x)
	}
}

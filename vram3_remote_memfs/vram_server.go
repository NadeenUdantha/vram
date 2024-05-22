package main

import (
	"fmt"
	"net/http"
	"unsafe"

	"github.com/fasthttp/websocket"
	"golang.org/x/sys/windows"
)

func init() {
	go vram_server()
}

func vram_server() {
	http.ListenAndServe(":7885", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.Header)
		ws, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		throw(err)
		var msg vram_msg
		for {
			msg.read(ws)
			switch msg.typ {
			case Alloc:
				addr, err := windows.VirtualAlloc(0, uintptr(msg.sz), windows.MEM_RESERVE|windows.MEM_COMMIT, windows.PAGE_EXECUTE_READWRITE)
				throw(err)
				msg.addr = uint64(addr)
				msg.write(ws)
			case Free:
				throw(windows.VirtualFree(uintptr(msg.addr), 0, windows.MEM_RELEASE))
				msg.write(ws)
			case Read:
				msg.data = unsafe.Slice((*byte)(unsafe.Pointer(uintptr(msg.addr))), msg.sz)
				msg.write(ws)
			case Write:
				copy(unsafe.Slice((*byte)(unsafe.Pointer(uintptr(msg.addr))), msg.sz), msg.data)
				msg.data = nil
				msg.write(ws)
			}
		}
	}))
}

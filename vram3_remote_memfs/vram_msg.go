package main

import (
	"encoding/binary"
	"fmt"

	"github.com/fasthttp/websocket"
)

const logging = false

type vram_msg struct {
	typ  VramMsgType
	addr uint64
	sz   uint64
	data []byte
}

func (msg *vram_msg) read(ws *websocket.Conn) {
	_, d, err := ws.ReadMessage()
	throw(err)
	msg.typ = VramMsgType(d[0])
	msg.addr = binary.LittleEndian.Uint64(d[1:])
	msg.sz = binary.LittleEndian.Uint64(d[1+8:])
	msg.data = d[1+8+8:]
	if logging {
		fmt.Println("msg read", msg.typ, msg.addr, msg.sz, len(msg.data))
	}
}

func (msg *vram_msg) write(ws *websocket.Conn) {
	d := make([]byte, 1+8+8+len(msg.data))
	d[0] = byte(msg.typ)
	binary.LittleEndian.PutUint64(d[1:], msg.addr)
	binary.LittleEndian.PutUint64(d[1+8:], msg.sz)
	copy(d[1+8+8:], msg.data)
	if logging {
		fmt.Println("msg write", msg.typ, msg.addr, msg.sz, len(msg.data))
	}
	throw(ws.WriteMessage(websocket.BinaryMessage, d))
}

type VramMsgType uint8

const (
	Alloc VramMsgType = 1
	Free  VramMsgType = 2
	Read  VramMsgType = 3
	Write VramMsgType = 4
)

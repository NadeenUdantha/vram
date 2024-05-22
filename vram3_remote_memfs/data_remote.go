package main

import (
	"github.com/fasthttp/websocket"
)

const (
	remote_addr = "ws://127.0.0.1:7885/"
	//remote_addr = "ws://192.168.1.39:7885/"
)

type data_remote struct {
	addr uint64
}

var ws *websocket.Conn

func vram_client() {
	var err error
	ws, _, err = websocket.DefaultDialer.Dial(remote_addr, nil)
	throw(err)
}

func new_data_remote() node_data {
	if ws == nil {
		vram_client()
	}
	return &data_remote{}
}

func (d *data_remote) read(b []byte, off, len int64) {
	if d.addr == 0 {
		return
	}
	msg := &vram_msg{
		typ:  Read,
		addr: d.addr + uint64(off),
		sz:   uint64(len),
	}
	msg.write(ws)
	msg.read(ws)
	assert(copy(b, msg.data) == int(len))
}

func (d *data_remote) write(b []byte, off, len int64) {
	assert(d.addr != 0)
	msg := &vram_msg{
		typ:  Write,
		addr: d.addr + uint64(off),
		sz:   uint64(len),
		data: b[:len],
	}
	msg.write(ws)
	msg.read(ws)
}

func (d *data_remote) resize(size int64) {
	assert(d.addr == 0)
	msg := &vram_msg{
		typ: Alloc,
		sz:  uint64(size),
	}
	msg.write(ws)
	msg.read(ws)
	d.addr = msg.addr
}

func (d *data_remote) close() {
	assert(d.addr == 0)
	msg := &vram_msg{
		typ:  Free,
		addr: d.addr,
	}
	msg.write(ws)
	msg.read(ws)
}

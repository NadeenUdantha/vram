package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

func new_data_impl(name string, node *node_t) node_data {
	//return new_data_bytes()
	return new_data_remote()
}

type node_data interface {
	resize(sz int64)
	read(b []byte, off, len int64)
	write(b []byte, off, len int64)
	close()
}

type node_data_log struct {
	name    string
	node    *node_t
	sz      uint64
	x       node_data
	discard bool
	written bool
}

func (r *node_data_log) rec() {
	if e := recover(); e != nil {
		r.logf("error: %v", e)
		debug.PrintStack()
	}
}

var _ = fmt.Printf

func (r *node_data_log) logf(z string, x ...interface{}) {
	//fmt.Printf(fmt.Sprintf("[data] %s %d %d ", r.name, r.node.stat.Ino, r.sz)+z+"\n", x...)
}

func (r *node_data_log) resize(sz int64) {
	defer r.rec()
	r.logf("resize %d", sz)
	r.sz = uint64(sz)
	if sz == 0 {
		return
	}
	r.x.resize(sz)
}

func (r *node_data_log) read(b []byte, off, len int64) {
	defer r.rec()
	if len == 0 {
		return
	}
	if !r.written {
		return
	}
	if !r.discard {
		r.logf("read %d %d", off, len)
		r.x.read(b, off, len)
	}
}

func (r *node_data_log) write(b []byte, off, len int64) {
	defer r.rec()
	if len == 0 {
		return
	}
	if !r.discard {
		r.logf("write %d %d", off, len)
		r.written = true
		r.x.write(b, off, len)
	}
}

func (r *node_data_log) close() {
	defer r.rec()
	r.logf("close")
	r.x.close()
}

func new_data_log(name string, node *node_t) *node_data_log {
	x := &node_data_log{
		name:    name,
		node:    node,
		sz:      0,
		x:       new_data_impl(name, node),
		discard: true,
	}
	if strings.HasSuffix(name, ".open") {
		node.parent.chld[strings.TrimSuffix(name, ".open")].data.discard = false
	} else if strings.HasSuffix(name, ".close") {
		node.parent.chld[strings.TrimSuffix(name, ".close")].data.discard = true
	}
	runtime.SetFinalizer(x, func(z node_data) {
		z.close()
		runtime.SetFinalizer(z, nil) //???
	})
	return x
}

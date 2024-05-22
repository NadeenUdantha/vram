package main

import (
	"sync"
)

type vrams struct {
	vs  map[uint32]*vram
	vsl sync.RWMutex
}

var vs *vrams

func init() {
	vs = &vrams{
		vs: make(map[uint32]*vram),
	}
}

func (vs *vrams) get(pid uint32) *vram {
	vs.vsl.RLock()
	defer vs.vsl.RUnlock()
	return vs.vs[pid]
}

func (vs *vrams) create(pid uint32, ph uintptr) *vram {
	vr := new_vram(pid, ph)
	vs.vsl.Lock()
	defer vs.vsl.Unlock()
	vs.vs[pid] = vr
	return vr
}

func (vs *vrams) remove(pid uint32) {
	vs.vsl.Lock()
	defer vs.vsl.Unlock()
	delete(vs.vs, pid)
}

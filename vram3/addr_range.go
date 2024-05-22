package main

type addr_range struct {
	as map[uint64]uint64
}

func new_addr_range() *addr_range {
	return &addr_range{
		as: make(map[uint64]uint64),
	}
}

func (ar *addr_range) in(addr, sz uint64) bool {
	for a, s := range ar.as {
		if a <= addr && addr+sz <= a+s {
			return true
		}
	}
	return false
}

func (ar *addr_range) add(addr, sz uint64) {
	ar.as[addr] = sz
}

func (ar *addr_range) remove(addr, sz uint64) {
	delete(ar.as, addr)
}

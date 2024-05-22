package main

type data_bytes struct {
	v []byte
}

func new_data_bytes() node_data {
	return &data_bytes{}
}

func (d *data_bytes) read(b []byte, off, len int64) {
	copy(b, d.v[off:][:len])
}

func (d *data_bytes) write(b []byte, off, len int64) {
	copy(d.v[off:][:len], b)
}

func (d *data_bytes) resize(size int64) {
	if int(size) == len(d.v) {
		return
	}
	x := make([]byte, size)
	copy(x, d.v)
	d.v = x
}

func (d *data_bytes) close() {
	d.v = nil
}

/*
 * memfs.go
 *
 * Copyright 2017-2020 Bill Zissimopoulos
 */
/*
 * This file is part of Cgofuse.
 *
 * It is licensed under the MIT license. The full license text can be found
 * in the License.txt file at the root of this project.
 */

//https://github.com/winfsp/cgofuse/blob/master/examples/memfs/memfs.go

package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/winfsp/cgofuse/fuse"
)

func trace(vals ...interface{}) func(vals ...interface{}) {
	uid, gid, _ := fuse.Getcontext()
	return Trace(1, fmt.Sprintf("[uid=%v,gid=%v]", uid, gid), vals...)
}

func split(path string) []string {
	return strings.Split(path, "/")
}

type node_t struct {
	stat    fuse.Stat_t
	xatr    map[string][]byte
	chld    map[string]*node_t
	data    *node_data_log
	opencnt int
	parent  *node_t
}

func newNode(name string, parent *node_t, dev uint64, ino uint64, mode uint32, uid uint32, gid uint32) *node_t {
	tmsp := fuse.Now()
	self := node_t{
		stat: fuse.Stat_t{
			Dev:      dev,
			Ino:      ino,
			Mode:     mode,
			Nlink:    1,
			Uid:      uid,
			Gid:      gid,
			Size:     0,
			Atim:     tmsp,
			Mtim:     tmsp,
			Ctim:     tmsp,
			Birthtim: tmsp,
			Flags:    0,
		},
		xatr:    nil,
		chld:    nil,
		opencnt: 0,
		parent:  parent,
	}
	if fuse.S_IFDIR == self.stat.Mode&fuse.S_IFMT {
		self.chld = map[string]*node_t{}
	}
	self.data = new_data_log(name, &self)
	return &self
}

type Memfs struct {
	fuse.FileSystemBase
	lock    sync.Mutex
	ino     uint64
	root    *node_t
	openmap map[uint64]*node_t
}

func (self *Memfs) Mknod(path string, mode uint32, dev uint64) (errc int) {
	defer trace(path, mode, dev)(&errc)
	defer self.synchronize()()
	return self.makeNode(path, mode, dev)
}

func (self *Memfs) Mkdir(path string, mode uint32) (errc int) {
	defer trace(path, mode)(&errc)
	defer self.synchronize()()
	return self.makeNode(path, fuse.S_IFDIR|(mode&07777), 0)
}

func (self *Memfs) Unlink(path string) (errc int) {
	defer trace(path)(&errc)
	defer self.synchronize()()
	return self.removeNode(path, false)
}

func (self *Memfs) Rmdir(path string) (errc int) {
	defer trace(path)(&errc)
	defer self.synchronize()()
	return self.removeNode(path, true)
}

func (self *Memfs) Link(oldpath string, newpath string) (errc int) {
	defer trace(oldpath, newpath)(&errc)
	defer self.synchronize()()
	_, _, oldnode := self.lookupNode(oldpath, nil)
	if nil == oldnode {
		return -fuse.ENOENT
	}
	newprnt, newname, newnode := self.lookupNode(newpath, nil)
	if nil == newprnt {
		return -fuse.ENOENT
	}
	if nil != newnode {
		return -fuse.EEXIST
	}
	oldnode.stat.Nlink++
	newprnt.chld[newname] = oldnode
	tmsp := fuse.Now()
	oldnode.stat.Ctim = tmsp
	newprnt.stat.Ctim = tmsp
	newprnt.stat.Mtim = tmsp
	return 0
}

func (self *Memfs) Rename(oldpath string, newpath string) (errc int) {
	defer trace(oldpath, newpath)(&errc)
	defer self.synchronize()()
	oldprnt, oldname, oldnode := self.lookupNode(oldpath, nil)
	if nil == oldnode {
		return -fuse.ENOENT
	}
	newprnt, newname, newnode := self.lookupNode(newpath, oldnode)
	if nil == newprnt {
		return -fuse.ENOENT
	}
	if "" == newname {
		// guard against directory loop creation
		return -fuse.EINVAL
	}
	if oldprnt == newprnt && oldname == newname {
		return 0
	}
	if nil != newnode {
		errc = self.removeNode(newpath, fuse.S_IFDIR == oldnode.stat.Mode&fuse.S_IFMT)
		if 0 != errc {
			return errc
		}
	}
	delete(oldprnt.chld, oldname)
	newprnt.chld[newname] = oldnode
	return 0
}

func (self *Memfs) Chmod(path string, mode uint32) (errc int) {
	defer trace(path, mode)(&errc)
	defer self.synchronize()()
	_, _, node := self.lookupNode(path, nil)
	if nil == node {
		return -fuse.ENOENT
	}
	node.stat.Mode = (node.stat.Mode & fuse.S_IFMT) | mode&07777
	node.stat.Ctim = fuse.Now()
	return 0
}

func (self *Memfs) Chown(path string, uid uint32, gid uint32) (errc int) {
	defer trace(path, uid, gid)(&errc)
	defer self.synchronize()()
	_, _, node := self.lookupNode(path, nil)
	if nil == node {
		return -fuse.ENOENT
	}
	if ^uint32(0) != uid {
		node.stat.Uid = uid
	}
	if ^uint32(0) != gid {
		node.stat.Gid = gid
	}
	node.stat.Ctim = fuse.Now()
	return 0
}

func (self *Memfs) Utimens(path string, tmsp []fuse.Timespec) (errc int) {
	defer trace(path, tmsp)(&errc)
	defer self.synchronize()()
	_, _, node := self.lookupNode(path, nil)
	if nil == node {
		return -fuse.ENOENT
	}
	node.stat.Ctim = fuse.Now()
	if nil == tmsp {
		tmsp0 := node.stat.Ctim
		tmsa := [2]fuse.Timespec{tmsp0, tmsp0}
		tmsp = tmsa[:]
	}
	node.stat.Atim = tmsp[0]
	node.stat.Mtim = tmsp[1]
	return 0
}

func (self *Memfs) Open(path string, flags int) (errc int, fh uint64) {
	defer trace(path, flags)(&errc, &fh)
	defer self.synchronize()()
	return self.openNode(path, false)
}

func (self *Memfs) Getattr(path string, stat *fuse.Stat_t, fh uint64) (errc int) {
	defer trace(path, fh)(&errc, stat)
	defer self.synchronize()()
	node := self.getNode(path, fh)
	if nil == node {
		return -fuse.ENOENT
	}
	*stat = node.stat
	return 0
}

func (self *Memfs) Truncate(path string, size int64, fh uint64) (errc int) {
	defer trace(path, size, fh)(&errc)
	defer self.synchronize()()
	node := self.getNode(path, fh)
	if nil == node {
		return -fuse.ENOENT
	}
	node.data.resize(size)
	node.stat.Size = size
	tmsp := fuse.Now()
	node.stat.Ctim = tmsp
	node.stat.Mtim = tmsp
	return 0
}

func (self *Memfs) Read(path string, buff []byte, ofst int64, fh uint64) (n int) {
	defer trace(path, buff, ofst, fh)(&n)
	defer self.synchronize()()
	node := self.getNode(path, fh)
	if nil == node {
		return -fuse.ENOENT
	}
	endofst := ofst + int64(len(buff))
	if endofst > node.stat.Size {
		endofst = node.stat.Size
	}
	if endofst < ofst {
		return 0
	}
	node.data.read(buff, ofst, endofst-ofst)
	n = int(endofst - ofst)
	node.stat.Atim = fuse.Now()
	return
}

func (self *Memfs) Write(path string, buff []byte, ofst int64, fh uint64) (n int) {
	defer trace(path, buff, ofst, fh)(&n)
	defer self.synchronize()()
	node := self.getNode(path, fh)
	if nil == node {
		return -fuse.ENOENT
	}
	endofst := ofst + int64(len(buff))
	if endofst > node.stat.Size {
		node.data.resize(endofst)
		node.stat.Size = endofst
	}
	node.data.write(buff, ofst, endofst-ofst)
	n = int(endofst - ofst)
	tmsp := fuse.Now()
	node.stat.Ctim = tmsp
	node.stat.Mtim = tmsp
	return
}

func (self *Memfs) Release(path string, fh uint64) (errc int) {
	defer trace(path, fh)(&errc)
	defer self.synchronize()()
	return self.closeNode(fh)
}

func (self *Memfs) Opendir(path string) (errc int, fh uint64) {
	defer trace(path)(&errc, &fh)
	defer self.synchronize()()
	return self.openNode(path, true)
}

func (self *Memfs) Readdir(path string,
	fill func(name string, stat *fuse.Stat_t, ofst int64) bool,
	ofst int64,
	fh uint64) (errc int) {
	defer trace(path, fill, ofst, fh)(&errc)
	defer self.synchronize()()
	node := self.openmap[fh]
	fill(".", &node.stat, 0)
	fill("..", nil, 0)
	for name, chld := range node.chld {
		if !fill(name, &chld.stat, 0) {
			break
		}
	}
	return 0
}

func (self *Memfs) Releasedir(path string, fh uint64) (errc int) {
	defer trace(path, fh)(&errc)
	defer self.synchronize()()
	return self.closeNode(fh)
}

func (self *Memfs) Setxattr(path string, name string, value []byte, flags int) (errc int) {
	defer trace(path, name, value, flags)(&errc)
	defer self.synchronize()()
	_, _, node := self.lookupNode(path, nil)
	if nil == node {
		return -fuse.ENOENT
	}
	if "com.apple.ResourceFork" == name {
		return -fuse.ENOTSUP
	}
	if fuse.XATTR_CREATE == flags {
		if _, ok := node.xatr[name]; ok {
			return -fuse.EEXIST
		}
	} else if fuse.XATTR_REPLACE == flags {
		if _, ok := node.xatr[name]; !ok {
			return -fuse.ENOATTR
		}
	}
	xatr := make([]byte, len(value))
	copy(xatr, value)
	if nil == node.xatr {
		node.xatr = map[string][]byte{}
	}
	node.xatr[name] = xatr
	return 0
}

func (self *Memfs) Getxattr(path string, name string) (errc int, xatr []byte) {
	defer trace(path, name)(&errc, &xatr)
	defer self.synchronize()()
	_, _, node := self.lookupNode(path, nil)
	if nil == node {
		return -fuse.ENOENT, nil
	}
	if "com.apple.ResourceFork" == name {
		return -fuse.ENOTSUP, nil
	}
	xatr, ok := node.xatr[name]
	if !ok {
		return -fuse.ENOATTR, nil
	}
	return 0, xatr
}

func (self *Memfs) Removexattr(path string, name string) (errc int) {
	defer trace(path, name)(&errc)
	defer self.synchronize()()
	_, _, node := self.lookupNode(path, nil)
	if nil == node {
		return -fuse.ENOENT
	}
	if "com.apple.ResourceFork" == name {
		return -fuse.ENOTSUP
	}
	if _, ok := node.xatr[name]; !ok {
		return -fuse.ENOATTR
	}
	delete(node.xatr, name)
	return 0
}

func (self *Memfs) Listxattr(path string, fill func(name string) bool) (errc int) {
	defer trace(path, fill)(&errc)
	defer self.synchronize()()
	_, _, node := self.lookupNode(path, nil)
	if nil == node {
		return -fuse.ENOENT
	}
	for name := range node.xatr {
		if !fill(name) {
			return -fuse.ERANGE
		}
	}
	return 0
}

func (self *Memfs) Chflags(path string, flags uint32) (errc int) {
	defer trace(path, flags)(&errc)
	defer self.synchronize()()
	_, _, node := self.lookupNode(path, nil)
	if nil == node {
		return -fuse.ENOENT
	}
	node.stat.Flags = flags
	node.stat.Ctim = fuse.Now()
	return 0
}

func (self *Memfs) Setcrtime(path string, tmsp fuse.Timespec) (errc int) {
	defer trace(path, tmsp)(&errc)
	defer self.synchronize()()
	_, _, node := self.lookupNode(path, nil)
	if nil == node {
		return -fuse.ENOENT
	}
	node.stat.Birthtim = tmsp
	node.stat.Ctim = fuse.Now()
	return 0
}

func (self *Memfs) Setchgtime(path string, tmsp fuse.Timespec) (errc int) {
	defer trace(path, tmsp)(&errc)
	defer self.synchronize()()
	_, _, node := self.lookupNode(path, nil)
	if nil == node {
		return -fuse.ENOENT
	}
	node.stat.Ctim = tmsp
	return 0
}

func (self *Memfs) lookupNode(path string, ancestor *node_t) (prnt *node_t, name string, node *node_t) {
	prnt = self.root
	name = ""
	node = self.root
	for _, c := range split(path) {
		if "" != c {
			if 255 < len(c) {
				panic(fuse.Error(-fuse.ENAMETOOLONG))
			}
			prnt, name = node, c
			if node == nil {
				return
			}
			node = node.chld[c]
			if nil != ancestor && node == ancestor {
				name = "" // special case loop condition
				return
			}
		}
	}
	return
}

func (self *Memfs) makeNode(path string, mode uint32, dev uint64) int {
	prnt, name, node := self.lookupNode(path, nil)
	if nil == prnt {
		return -fuse.ENOENT
	}
	if nil != node {
		return -fuse.EEXIST
	}
	self.ino++
	uid, gid, _ := fuse.Getcontext()
	node = newNode(name, prnt, dev, self.ino, mode, uid, gid)
	prnt.chld[name] = node
	prnt.stat.Ctim = node.stat.Ctim
	prnt.stat.Mtim = node.stat.Ctim
	return 0
}

func (self *Memfs) removeNode(path string, dir bool) int {
	prnt, name, node := self.lookupNode(path, nil)
	if nil == node {
		return -fuse.ENOENT
	}
	if !dir && fuse.S_IFDIR == node.stat.Mode&fuse.S_IFMT {
		return -fuse.EISDIR
	}
	if dir && fuse.S_IFDIR != node.stat.Mode&fuse.S_IFMT {
		return -fuse.ENOTDIR
	}
	if 0 < len(node.chld) {
		return -fuse.ENOTEMPTY
	}
	node.stat.Nlink--
	delete(prnt.chld, name)
	tmsp := fuse.Now()
	node.stat.Ctim = tmsp
	prnt.stat.Ctim = tmsp
	prnt.stat.Mtim = tmsp
	return 0
}

func (self *Memfs) openNode(path string, dir bool) (int, uint64) {
	_, _, node := self.lookupNode(path, nil)
	if nil == node {
		return -fuse.ENOENT, ^uint64(0)
	}
	if !dir && fuse.S_IFDIR == node.stat.Mode&fuse.S_IFMT {
		return -fuse.EISDIR, ^uint64(0)
	}
	if dir && fuse.S_IFDIR != node.stat.Mode&fuse.S_IFMT {
		return -fuse.ENOTDIR, ^uint64(0)
	}
	node.opencnt++
	if 1 == node.opencnt {
		self.openmap[node.stat.Ino] = node
	}
	return 0, node.stat.Ino
}

func (self *Memfs) closeNode(fh uint64) int {
	node := self.openmap[fh]
	node.opencnt--
	if 0 == node.opencnt {
		delete(self.openmap, node.stat.Ino)
	}
	return 0
}

func (self *Memfs) getNode(path string, fh uint64) *node_t {
	if ^uint64(0) == fh {
		_, _, node := self.lookupNode(path, nil)
		return node
	} else {
		return self.openmap[fh]
	}
}

func (self *Memfs) synchronize() func() {
	self.lock.Lock()
	return func() {
		self.lock.Unlock()
	}
}

func NewMemfs() *Memfs {
	self := Memfs{}
	defer self.synchronize()()
	self.ino++
	self.root = newNode("/", nil, 0, self.ino, fuse.S_IFDIR|00777, 0, 0)
	self.openmap = map[uint64]*node_t{}
	return &self
}

var _ fuse.FileSystemChflags = (*Memfs)(nil)
var _ fuse.FileSystemSetcrtime = (*Memfs)(nil)
var _ fuse.FileSystemSetchgtime = (*Memfs)(nil)

func init() {
	go func() {
		memfs := NewMemfs()
		host := fuse.NewFileSystemHost(memfs)
		host.SetCapReaddirPlus(true)
		host.Mount("Z:", []string{
			//"-s",
			//"-d",
			//"-o debug",
			"-ouid=-1,gid=-1",
		})
	}()
}

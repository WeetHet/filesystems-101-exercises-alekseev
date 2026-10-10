package ptr

import (
	"unsafe"
)

// Malloc returns an uninitialised slice that has size bytes.
func Malloc(size uint) []byte {
	bytes := (*byte)(mallocgc(uintptr(size), 0, false))
	return unsafe.Slice(bytes, size)
}

//go:linkname mallocgc runtime.mallocgc
func mallocgc(size uintptr, typ uintptr, needzero bool) unsafe.Pointer

// Package mmio provides typed volatile memory-mapped register access.
package mmio

import "unsafe"

// Register32 is a 32-bit memory-mapped register address.
type Register32 uintptr

//renvo:load
func Load32(address uintptr) uint32 {
	return *(*uint32)(unsafe.Pointer(address))
}

//renvo:store
func Store32(address uintptr, value uint32) {
	*(*uint32)(unsafe.Pointer(address)) = value
}

//renvo:load
func Load8(address uintptr) uint8 {
	return *(*uint8)(unsafe.Pointer(address))
}

//renvo:store
func Store8(address uintptr, value uint8) {
	*(*uint8)(unsafe.Pointer(address)) = value
}

// Load performs a volatile register read.
func (r Register32) Load() uint32 { return Load32(uintptr(r)) }

// Store performs a volatile register write.
func (r Register32) Store(value uint32) { Store32(uintptr(r), value) }

// Set writes value to a write-one-to-set alias register.
func (r Register32) Set(value uint32) { r.Store(value) }

// Clear writes value to a write-one-to-clear alias register.
func (r Register32) Clear(value uint32) { r.Store(value) }

// Replace performs a conventional read/modify/write operation.
func (r Register32) Replace(clear, set uint32) {
	r.Store(r.Load()&^clear | set)
}

package mmio_test

import "renvo.dev/device/mmio"

func ExampleLoad32() {
	const statusAddress = uintptr(0x40000020)
	status := mmio.Load32(statusAddress)
	ready := status&(1<<7) != 0
	_ = ready
}

func ExampleStore32() {
	const commandAddress = uintptr(0x40000024)
	mmio.Store32(commandAddress, 1<<3)
}

func ExampleRegister32_Load() {
	status := mmio.Register32(0x40000020)
	if status.Load()&(1<<7) != 0 {
		// The peripheral is ready.
	}
}

func ExampleRegister32_Store() {
	control := mmio.Register32(0x40000010)
	control.Store(1<<0 | 2<<4)
}

// Set performs a read-modify-write that preserves unrelated control bits.
func ExampleRegister32_Set() {
	control := mmio.Register32(0x40000010)
	control.Set(1 << 8)
}

// Clear performs a read-modify-write that preserves unrelated control bits.
func ExampleRegister32_Clear() {
	control := mmio.Register32(0x40000010)
	control.Clear(1 << 8)
}

// Replace changes one bitfield without disturbing the rest of the register.
func ExampleRegister32_Replace() {
	control := mmio.Register32(0x40000010)
	const modeMask = uint32(0b11 << 4)
	control.Replace(modeMask, 0b01<<4)
}

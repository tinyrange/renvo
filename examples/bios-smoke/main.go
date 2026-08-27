package main

import "renvo.dev/device/bios"

func fail(code byte) {
	bios.Out8(0xf4, code)
	for {
	}
}

func mark(value byte) { bios.Out8(0xe9, value) }

func serialWrite(value byte) {
	for bios.In8(0x3fd)&0x20 == 0 {
	}
	bios.Out8(0x3f8, value)
}

func main() {
	mark('A')
	print("PASS\n")
	mark('B')
	drive := bios.BootDrive()
	mark('C')
	var err error
	if bios.ExtensionsAvailable(drive) {
		err = bios.ReadSectors(drive, 0, 1, 0x2000, 0)
	} else {
		err = bios.ReadCHS(drive, 0, 0, 1, 1, 0x2000, 0)
	}
	if err != nil {
		if code, ok := err.(bios.Error); ok {
			fail(byte(code) + 16)
		}
		fail(4)
	}
	mark('D')
	if bios.Load8(0x2000, 510) != 0x55 {
		fail(5)
	}
	if bios.Load8(0x2000, 511) != 0xaa {
		fail(8)
	}
	bios.Store8(0x2000, 0, 0x5a)
	if bios.Load8(0x2000, 0) != 0x5a {
		fail(9)
	}
	bios.Write(0x2000, 1, []byte{0x12, 0x34})
	if bios.Load8(0x2000, 1) != 0x12 || bios.Load8(0x2000, 2) != 0x34 {
		fail(10)
	}
	mark('E')
	bios.Ticks()
	mark('F')
	if err := bios.InitSerial(0, 0xe3); err != nil {
		fail(6)
	}
	mark('G')
	// Program COM1 explicitly after exercising the firmware initializer. This
	// keeps the smoke oracle independent of BIOS transmit timeout policy.
	bios.Out8(0x3f9, 0)
	bios.Out8(0x3fb, 0x80)
	bios.Out8(0x3f8, 1)
	bios.Out8(0x3f9, 0)
	bios.Out8(0x3fb, 3)
	bios.Out8(0x3fa, 0xc7)
	bios.Out8(0x3fc, 0x0b)
	for _, value := range []byte("PASS\r\n") {
		serialWrite(value)
	}
	mark('H')
	bios.Out8(0xf4, 42)
}

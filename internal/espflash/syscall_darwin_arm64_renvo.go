//go:build renvo && darwin && arm64

package espflash

import "unsafe"

// Apple's public open is variadic when O_CREAT supplies a mode. Use its
// fixed-signature syscall wrapper for the same arm64 ABI reason as ioctl.
// renvo:linkstatic /usr/lib/libSystem.B.dylib,___open
func darwinOpen(path *byte, flags int, mode int) int { return -1 }

// renvo:linkstatic /usr/lib/libSystem.B.dylib,close
func darwinClose(fd int) int { return -1 }

// renvo:linkstatic /usr/lib/libSystem.B.dylib,read
func darwinRead(fd int, data *byte, size int) int { return -1 }

// renvo:linkstatic /usr/lib/libSystem.B.dylib,write
func darwinWrite(fd int, data *byte, size int) int { return -1 }

// Apple's public ioctl is variadic, so its third argument uses the arm64
// variadic ABI. Link the fixed-signature syscall wrapper instead.
// renvo:linkstatic /usr/lib/libSystem.B.dylib,___ioctl
func darwinIoctl(fd int, request uint64, pointer *byte) int { return -1 }

// renvo:linkstatic /usr/lib/libSystem.B.dylib,usleep
func darwinUsleep(microseconds int) int { return -1 }

// renvo:linkstatic /usr/lib/libSystem.B.dylib,tcgetattr
func darwinTCGetAttr(fd int, state *byte) int { return -1 }

// renvo:linkstatic /usr/lib/libSystem.B.dylib,tcsetattr
func darwinTCSetAttr(fd int, action int, state *byte) int { return -1 }

// renvo:linkstatic /usr/lib/libSystem.B.dylib,cfmakeraw
func darwinMakeRaw(state *byte) {}

// renvo:linkstatic /usr/lib/libSystem.B.dylib,cfsetspeed
func darwinSetSpeed(state *byte, speed int) int { return -1 }

// renvo:linkstatic /usr/lib/libSystem.B.dylib,___error
func darwinErrorPointer() *int32 { return nil }

func serialPortPrefixes() []string { return []string{"cu.usbmodem", "cu.usbserial"} }
func hostReadFlags() int           { return 0 }
func hostSerialFlags() int         { return 2 | 0x20000 | 0x4 }
func hostCreateFlags() int         { return 2 | 0x200 | 0x400 }
func hostDTR() int                 { return 0x002 }
func hostRTS() int                 { return 0x004 }
func hostModemGet() int            { return 0x4004746a }
func hostModemWrite() int          { return 0x8004746d }
func hostModemSet() int            { return 0x8004746c }
func hostModemClear() int          { return 0x8004746b }
func hostClose(fd int) int         { return darwinClose(fd) }
func hostError(result int) int {
	pointer := darwinErrorPointer()
	if pointer == nil {
		return -1
	}
	return int(*pointer)
}
func hostWouldBlock() int           { return 35 }
func hostPointer(pointer *byte) int { return int(unsafe.Pointer(pointer)) }
func hostIoctl(fd int, request int, pointer int) int {
	return darwinIoctl(fd, uint64(request), (*byte)(unsafe.Pointer(uintptr(pointer))))
}
func hostRead(fd int, pointer int, size int) int {
	return darwinRead(fd, (*byte)(unsafe.Pointer(uintptr(pointer))), size)
}
func hostWrite(fd int, pointer int, size int) int {
	return darwinWrite(fd, (*byte)(unsafe.Pointer(uintptr(pointer))), size)
}
func hostOpen(path string, flags int, mode int) int {
	name := append([]byte(path), 0)
	return darwinOpen(&name[0], flags, mode)
}

func configureSerial(port *serial) int {
	termios := make([]byte, 72)
	if darwinTCGetAttr(port.fd, &termios[0]) < 0 {
		return -1
	}
	darwinMakeRaw(&termios[0])
	put32(termios, 16, int(u32(termios, 16))|0x800|0x8000) // CREAD | CLOCAL
	if darwinSetSpeed(&termios[0], 115200) < 0 {
		return -1
	}
	termios[32+16] = 0 // VMIN
	termios[32+17] = 1 // VTIME: 100 ms
	return darwinTCSetAttr(port.fd, 0, &termios[0])
}

func sleep(milliseconds int) { darwinUsleep(milliseconds * 1000) }

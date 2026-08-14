//go:build renvo && linux && amd64

package main

import "unsafe"

const (
	sysRead      = 0
	sysWrite     = 1
	sysOpen      = 2
	sysClose     = 3
	sysIoctl     = 16
	sysNanosleep = 35
)

func syscall(number int, first int, second int, third int, fourth int, fifth int, sixth int) int {
	return 0
}

func serialPortPrefixes() []string { return []string{"ttyACM", "ttyUSB"} }
func hostReadFlags() int           { return 0 }
func hostSerialFlags() int         { return 2 | 0x100 | 0x800 }
func hostCreateFlags() int         { return 2 | 64 | 512 }
func hostDTR() int                 { return 0x002 }
func hostRTS() int                 { return 0x004 }
func hostModemGet() int            { return 0x5415 }
func hostModemWrite() int          { return 0x5418 }
func hostModemSet() int            { return 0x5416 }
func hostModemClear() int          { return 0x5417 }
func hostClose(fd int) int         { return syscall(sysClose, fd, 0, 0, 0, 0, 0) }
func hostError(result int) int {
	// Renvo's Linux syscall intrinsic returns the kernel result directly, so
	// failures are -errno rather than libc's -1 plus a thread-local errno.
	return -result
}
func hostWouldBlock() int { return 11 }
func hostIoctl(fd int, request int, pointer int) int {
	return syscall(sysIoctl, fd, request, pointer, 0, 0, 0)
}
func hostRead(fd int, pointer int, size int) int {
	return syscall(sysRead, fd, pointer, size, 0, 0, 0)
}
func hostWrite(fd int, pointer int, size int) int {
	return syscall(sysWrite, fd, pointer, size, 0, 0, 0)
}
func hostOpen(path string, flags int, mode int) int {
	name := append([]byte(path), 0)
	return syscall(sysOpen, int(unsafe.Pointer(&name[0])), flags, mode, 0, 0, 0)
}

func configureSerial(port *serial) int {
	termios := make([]byte, 64)
	if port.ioctl(0x5401, termios) < 0 {
		return -1
	}
	put32(termios, 0, 4)                      // IGNPAR
	put32(termios, 4, 0)                      // no output processing
	put32(termios, 8, 0x1002|0x30|0x80|0x800) // B115200, CS8, CREAD, CLOCAL
	put32(termios, 12, 0)                     // raw local mode
	termios[17+5] = 1                         // VTIME: 100 ms
	termios[17+6] = 0                         // VMIN
	return port.ioctl(0x5402, termios)
}

func sleep(milliseconds int) {
	timespec := make([]byte, 16)
	put64(timespec, 0, int64(milliseconds/1000))
	put64(timespec, 8, int64(milliseconds%1000)*1000000)
	syscall(sysNanosleep, int(unsafe.Pointer(&timespec[0])), 0, 0, 0, 0, 0)
}

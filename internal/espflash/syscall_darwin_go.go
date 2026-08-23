//go:build !renvo && darwin

package espflash

import (
	hostsyscall "syscall"
	"time"
	"unsafe"
)

func darwinSyscall(number uintptr, first uintptr, second uintptr, third uintptr) int {
	result, _, callErr := hostsyscall.Syscall(number, first, second, third)
	if callErr != 0 {
		return -int(callErr)
	}
	return int(result)
}

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
func hostError(result int) int     { return -result }
func hostWouldBlock() int          { return 35 }
func hostPointer(pointer *byte) int {
	return int(uintptr(unsafe.Pointer(pointer)))
}
func hostClose(fd int) int {
	return darwinSyscall(hostsyscall.SYS_CLOSE, uintptr(fd), 0, 0)
}
func hostIoctl(fd int, request int, pointer int) int {
	return darwinSyscall(hostsyscall.SYS_IOCTL, uintptr(fd), uintptr(request), uintptr(pointer))
}
func hostRead(fd int, pointer int, size int) int {
	return darwinSyscall(hostsyscall.SYS_READ, uintptr(fd), uintptr(pointer), uintptr(size))
}
func hostWrite(fd int, pointer int, size int) int {
	return darwinSyscall(hostsyscall.SYS_WRITE, uintptr(fd), uintptr(pointer), uintptr(size))
}
func hostOpen(path string, flags int, mode int) int {
	name := append([]byte(path), 0)
	return darwinSyscall(hostsyscall.SYS_OPEN, uintptr(unsafe.Pointer(&name[0])), uintptr(flags), uintptr(mode))
}

func configureSerial(port *serial) int {
	var state hostsyscall.Termios
	if hostIoctl(port.fd, hostsyscall.TIOCGETA, int(uintptr(unsafe.Pointer(&state)))) < 0 {
		return -1
	}
	state.Iflag &^= hostsyscall.IGNBRK | hostsyscall.BRKINT | hostsyscall.PARMRK |
		hostsyscall.ISTRIP | hostsyscall.INLCR | hostsyscall.IGNCR |
		hostsyscall.ICRNL | hostsyscall.IXON
	state.Oflag &^= hostsyscall.OPOST
	state.Lflag &^= hostsyscall.ECHO | hostsyscall.ECHONL | hostsyscall.ICANON |
		hostsyscall.ISIG | hostsyscall.IEXTEN
	state.Cflag &^= hostsyscall.CSIZE | hostsyscall.PARENB
	state.Cflag |= hostsyscall.CS8 | hostsyscall.CREAD | hostsyscall.CLOCAL
	state.Cc[hostsyscall.VMIN] = 0
	state.Cc[hostsyscall.VTIME] = 1
	state.Ispeed = hostsyscall.B115200
	state.Ospeed = hostsyscall.B115200
	return hostIoctl(port.fd, hostsyscall.TIOCSETA, int(uintptr(unsafe.Pointer(&state))))
}

func sleep(milliseconds int) { time.Sleep(time.Duration(milliseconds) * time.Millisecond) }

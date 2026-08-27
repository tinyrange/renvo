//go:build !renvo || !bios || !i8086

package bios

var InterruptHook func(vector byte, regs *Registers)
var PortInHook func(port uint16) byte
var PortOutHook func(port uint16, value uint16, width int)
var BootDriveHook func() byte

func hostInterrupt(vector byte, regs *Registers) {
	if InterruptHook != nil {
		InterruptHook(vector, regs)
	}
}
func interrupt10(regs *Registers) { hostInterrupt(0x10, regs) }
func interrupt13(regs *Registers) { hostInterrupt(0x13, regs) }
func interrupt14(regs *Registers) { hostInterrupt(0x14, regs) }
func interrupt15(regs *Registers) { hostInterrupt(0x15, regs) }
func interrupt16(regs *Registers) { hostInterrupt(0x16, regs) }
func interrupt17(regs *Registers) { hostInterrupt(0x17, regs) }
func interrupt1A(regs *Registers) { hostInterrupt(0x1a, regs) }
func portIn8(port uint16) byte {
	if PortInHook != nil {
		return PortInHook(port)
	}
	return 0
}
func portOut8(port uint16, value byte) {
	if PortOutHook != nil {
		PortOutHook(port, uint16(value), 1)
	}
}
func portOut16(port uint16, value uint16) {
	if PortOutHook != nil {
		PortOutHook(port, value, 2)
	}
}
func segmentLoad8(segment, offset uint16) byte         { return 0 }
func segmentStore8(segment, offset uint16, value byte) {}
func segmentWrite(segment, offset uint16, data []byte) {}
func bootDrive() byte {
	if BootDriveHook != nil {
		return BootDriveHook()
	}
	return 0x80
}

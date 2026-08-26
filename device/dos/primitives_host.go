//go:build !renvo || !msdos || !i8086

package dos

var InterruptHook func(vector byte, regs *Registers)
var PortInHook func(port uint16) byte
var PortOutHook func(port uint16, value uint16, width int)

func hostInterrupt(vector byte, regs *Registers) {
	if InterruptHook != nil {
		InterruptHook(vector, regs)
	}
}
func interrupt10(regs *Registers)   { hostInterrupt(0x10, regs) }
func interrupt13(regs *Registers)   { hostInterrupt(0x13, regs) }
func interrupt14(regs *Registers)   { hostInterrupt(0x14, regs) }
func interrupt15(regs *Registers)   { hostInterrupt(0x15, regs) }
func interrupt16(regs *Registers)   { hostInterrupt(0x16, regs) }
func interrupt17(regs *Registers)   { hostInterrupt(0x17, regs) }
func interrupt1A(regs *Registers)   { hostInterrupt(0x1a, regs) }
func interrupt21(regs *Registers)   { hostInterrupt(0x21, regs) }
func interrupt21ES(regs *Registers) { hostInterrupt(0x21, regs) }
func interrupt33(regs *Registers)   { hostInterrupt(0x33, regs) }
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
func segmentLoad8(segment, offset uint16) byte                      { return 0 }
func segmentStore8(segment, offset uint16, value byte)              {}
func segmentFill8(segment, offset uint16, value byte, count uint16) {}
func segmentWrite(segment, offset uint16, data []byte)              {}

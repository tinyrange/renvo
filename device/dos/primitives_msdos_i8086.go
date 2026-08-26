//go:build renvo && msdos && i8086

package dos

func interrupt10(regs *Registers)
func interrupt13(regs *Registers)
func interrupt14(regs *Registers)
func interrupt15(regs *Registers)
func interrupt16(regs *Registers)
func interrupt17(regs *Registers)
func interrupt1A(regs *Registers)
func interrupt21(regs *Registers)
func interrupt21ES(regs *Registers)
func interrupt33(regs *Registers)
func portIn8(port uint16) byte
func portOut8(port uint16, value byte)
func portOut16(port uint16, value uint16)
func segmentLoad8(segment, offset uint16) byte
func segmentStore8(segment, offset uint16, value byte)
func segmentFill8(segment, offset uint16, value byte, count uint16)
func segmentWrite(segment, offset uint16, data []byte)

//go:build renvo && bios && i8086

package bios

func interrupt10(regs *Registers)
func interrupt13(regs *Registers)
func interrupt14(regs *Registers)
func interrupt15(regs *Registers)
func interrupt16(regs *Registers)
func interrupt17(regs *Registers)
func interrupt1A(regs *Registers)
func portIn8(port uint16) byte
func portOut8(port uint16, value byte)
func portOut16(port uint16, value uint16)
func segmentLoad8(segment, offset uint16) byte
func segmentStore8(segment, offset uint16, value byte)
func segmentWrite(segment, offset uint16, data []byte)
func bootDrive() byte

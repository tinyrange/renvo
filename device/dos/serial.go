package dos

const (
	Serial110   = 0 << 5
	Serial150   = 1 << 5
	Serial300   = 2 << 5
	Serial600   = 3 << 5
	Serial1200  = 4 << 5
	Serial2400  = 5 << 5
	Serial4800  = 6 << 5
	Serial9600  = 7 << 5
	Serial8Bits = 3
)

type Serial struct{ Port byte }

func (s Serial) Configure(config byte) byte {
	regs := Registers{AX: uint16(config), DX: uint16(s.Port)}
	interrupt14(&regs)
	return byte(regs.AX >> 8)
}
func (s Serial) Write(value byte) byte {
	regs := Registers{AX: 0x0100 | uint16(value), DX: uint16(s.Port)}
	interrupt14(&regs)
	return byte(regs.AX >> 8)
}
func (s Serial) Read() (byte, byte) {
	regs := Registers{AX: 0x0200, DX: uint16(s.Port)}
	interrupt14(&regs)
	return byte(regs.AX), byte(regs.AX >> 8)
}
func (s Serial) Status() (line, modem byte) {
	regs := Registers{AX: 0x0300, DX: uint16(s.Port)}
	interrupt14(&regs)
	return byte(regs.AX >> 8), byte(regs.AX)
}

type Printer struct{ Port byte }

func (p Printer) Write(value byte) byte {
	regs := Registers{AX: uint16(value), DX: uint16(p.Port)}
	interrupt17(&regs)
	return byte(regs.AX >> 8)
}
func (p Printer) Status() byte {
	regs := Registers{AX: 0x0200, DX: uint16(p.Port)}
	interrupt17(&regs)
	return byte(regs.AX >> 8)
}

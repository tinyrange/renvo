package dos

type Key struct {
	ASCII byte
	Scan  byte
}

func KeyAvailable() bool {
	regs := Registers{AX: 0x0100}
	interrupt16(&regs)
	return regs.Flags&FlagZero == 0
}

func ReadKey() Key {
	regs := Registers{}
	interrupt16(&regs)
	return Key{ASCII: byte(regs.AX), Scan: byte(regs.AX >> 8)}
}

type Mouse struct {
	Buttons uint16
}

func OpenMouse() (*Mouse, bool) {
	regs := Registers{}
	interrupt33(&regs)
	if regs.AX == 0 {
		return nil, false
	}
	return &Mouse{Buttons: regs.BX}, true
}
func (m *Mouse) Show() { regs := Registers{AX: 1}; interrupt33(&regs) }
func (m *Mouse) Hide() { regs := Registers{AX: 2}; interrupt33(&regs) }
func (m *Mouse) Position() (x, y int, buttons uint16) {
	regs := Registers{AX: 3}
	interrupt33(&regs)
	return int(regs.CX), int(regs.DX), regs.BX
}
func (m *Mouse) SetBounds(minX, maxX, minY, maxY int) {
	horizontal := Registers{AX: 7, CX: uint16(minX), DX: uint16(maxX)}
	interrupt33(&horizontal)
	vertical := Registers{AX: 8, CX: uint16(minY), DX: uint16(maxY)}
	interrupt33(&vertical)
}

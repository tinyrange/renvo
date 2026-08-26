package dos

const (
	ModeText80x25 = 0x03
	ModeVGAPlanar = 0x12
	ModeVGA13     = 0x13
	vgaSegment    = 0xa000
)

func SetVideoMode(mode byte) {
	regs := Registers{AX: uint16(mode)}
	interrupt10(&regs)
}

func VideoMode() byte {
	regs := Registers{AX: 0x0f00}
	interrupt10(&regs)
	return byte(regs.AX)
}

func SetCursor(page byte, x, y int) {
	regs := Registers{AX: 0x0200, BX: uint16(page) << 8, DX: uint16(y)<<8 | uint16(x)}
	interrupt10(&regs)
}

func Cursor(page byte) (x, y int) {
	regs := Registers{AX: 0x0300, BX: uint16(page) << 8}
	interrupt10(&regs)
	return int(regs.DX & 255), int(regs.DX >> 8)
}

func Teletype(page, color, value byte) {
	regs := Registers{AX: 0x0e00 | uint16(value), BX: uint16(page)<<8 | uint16(color)}
	interrupt10(&regs)
}

type VGA13 struct{}

func OpenVGA13() *VGA13 { SetVideoMode(ModeVGA13); return &VGA13{} }
func (v *VGA13) Close() { SetVideoMode(ModeText80x25) }
func (v *VGA13) Set(x, y int, color byte) {
	if x >= 0 && x < 320 && y >= 0 && y < 200 {
		segmentStore8(vgaSegment, uint16(y)*320+uint16(x), color)
	}
}
func (v *VGA13) Clear(color byte) { segmentFill8(vgaSegment, 0, color, uint16(64000)) }
func (v *VGA13) Blit(offset uint16, pixels []byte) {
	if offset >= 64000 || len(pixels) == 0 {
		return
	}
	remaining := uint32(64000) - uint32(offset)
	if uint32(len(pixels)) > remaining {
		pixels = pixels[:int(remaining)]
	}
	segmentWrite(vgaSegment, offset, pixels)
}
func (v *VGA13) Palette(index byte, red, green, blue byte) {
	portOut8(0x3c8, index)
	portOut8(0x3c9, red>>2)
	portOut8(0x3c9, green>>2)
	portOut8(0x3c9, blue>>2)
}

func WaitVerticalRetrace() {
	for portIn8(0x3da)&8 != 0 {
	}
	for portIn8(0x3da)&8 == 0 {
	}
}

type VGAPlanar struct{}

func OpenVGAPlanar() *VGAPlanar { SetVideoMode(ModeVGAPlanar); return &VGAPlanar{} }
func (v *VGAPlanar) Close()     { SetVideoMode(ModeText80x25) }
func (v *VGAPlanar) Set(x, y int, color byte) {
	if x < 0 || x >= 640 || y < 0 || y >= 480 {
		return
	}
	portOut16(0x3ce, 0x0205)
	portOut16(0x3ce, uint16(0x08)|uint16(0x80>>uint(x&7))<<8)
	offset := uint16(y)*80 + uint16(x/8)
	_ = segmentLoad8(vgaSegment, offset)
	segmentStore8(vgaSegment, offset, color&15)
	portOut16(0x3ce, 0x0005)
	portOut16(0x3ce, 0xff08)
}

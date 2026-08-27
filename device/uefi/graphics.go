package uefi

import "unsafe"

var GraphicsOutputProtocolGUID = GUID{0x4a3823dc9042a9de, 0x6a5180d0de7afb96}

const (
	PixelRedGreenBlueReserved8BitPerColor = 0
	PixelBlueGreenRedReserved8BitPerColor = 1
	PixelBitMask                          = 2
	PixelBltOnly                          = 3
)

type GraphicsOutputProtocol struct {
	QueryMode uintptr
	SetMode   uintptr
	Blt       uintptr
	Mode      uintptr
}

func GraphicsOutput() (*GraphicsOutputProtocol, Status) {
	protocol, status := LocateProtocol(&GraphicsOutputProtocolGUID)
	return (*GraphicsOutputProtocol)(unsafe.Pointer(protocol)), status
}

func (g *GraphicsOutputProtocol) SetDisplayMode(mode uint32) Status {
	if g == nil {
		return InvalidParameter
	}
	return Status(call2(g.SetMode, pointer(unsafe.Pointer(g)), uintptr(mode)))
}

type Framebuffer struct {
	Address     uintptr
	Width       uint32
	Height      uint32
	Stride      uint32
	PixelFormat uint32
}

func (g *GraphicsOutputProtocol) Framebuffer() (Framebuffer, Status) {
	if g == nil || g.Mode == 0 {
		return Framebuffer{}, NotReady
	}
	info := loadWord(g.Mode, 8)
	if info == 0 {
		return Framebuffer{}, NotReady
	}
	format := load32(info, 12)
	if format != PixelRedGreenBlueReserved8BitPerColor &&
		format != PixelBlueGreenRedReserved8BitPerColor {
		return Framebuffer{}, Unsupported
	}
	return Framebuffer{loadWord(g.Mode, 24), load32(info, 4),
		load32(info, 8), load32(info, 32), format}, Success
}

func (f Framebuffer) Set(x, y int, red, green, blue byte) {
	if x < 0 || y < 0 || x >= int(f.Width) || y >= int(f.Height) {
		return
	}
	pixel := (*uint32)(unsafe.Pointer(f.Address + uintptr((y*int(f.Stride)+x)*4)))
	if f.PixelFormat == PixelRedGreenBlueReserved8BitPerColor {
		*pixel = uint32(red) | uint32(green)<<8 | uint32(blue)<<16
	} else {
		*pixel = uint32(blue) | uint32(green)<<8 | uint32(red)<<16
	}
}

func (f Framebuffer) Fill(red, green, blue byte) {
	for y := 0; y < int(f.Height); y++ {
		for x := 0; x < int(f.Width); x++ {
			f.Set(x, y, red, green, blue)
		}
	}
}

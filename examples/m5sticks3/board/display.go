package board

import (
	"unsafe"

	"renvo.dev/std/graphics"
)

const (
	DisplayWidth  = 135
	DisplayHeight = 240
)

const (
	lcdMOSI           = 39
	lcdClock          = 40
	lcdChipSelect     = 41
	lcdDataCommand    = 45
	lcdReset          = 21
	lcdBlack          = uint16(0x0000)
	lcdWhite          = uint16(0xffff)
	lcdRed            = uint16(0xf800)
	lcdYellow         = uint16(0xffe0)
	lcdGreen          = uint16(0x07e0)
	lcdCyan           = uint16(0x07ff)
	lcdBlue           = uint16(0x001f)
	lcdMagenta        = uint16(0xf81f)
	spi3Base          = uintptr(0x60025000)
	spi3Command       = spi3Base + 0x00
	spi3Control       = spi3Base + 0x08
	spi3Clock         = spi3Base + 0x0c
	spi3User          = spi3Base + 0x10
	spi3DataLength    = spi3Base + 0x1c
	spi3Misc          = spi3Base + 0x20
	spi3DMAConfig     = spi3Base + 0x30
	spi3DMAIntClear   = spi3Base + 0x38
	spi3DMAIntRaw     = spi3Base + 0x3c
	spi3Data          = spi3Base + 0x98
	spi3ClockGate     = spi3Base + 0xe8
	systemClock0      = uintptr(0x600c0018)
	systemClock1      = uintptr(0x600c001c)
	systemReset0      = uintptr(0x600c0020)
	systemReset1      = uintptr(0x600c0024)
	gdmaBase          = uintptr(0x6003f000)
	gdmaOutConfig0    = gdmaBase + 0x60
	gdmaOutConfig1    = gdmaBase + 0x64
	gdmaOutIntRaw     = gdmaBase + 0x68
	gdmaOutIntClear   = gdmaBase + 0x74
	gdmaOutLink       = gdmaBase + 0x80
	gdmaOutPeripheral = gdmaBase + 0xa8
	gdmaMiscConfig    = gdmaBase + 0x3c8
	spi3Enable        = uint32(1 << 16)
	gdmaEnable        = uint32(1 << 6)
	spi3Update        = uint32(1 << 23)
	spi3UserStart     = uint32(1 << 24)
	spi3UserMOSI      = uint32(1 << 27)
	spi3DMATX         = uint32(1 << 28)
	gdmaOutLinkParked = uint32(1 << 23)
)

type dmaDescriptor struct {
	control uint32
	buffer  uintptr
	next    uintptr
}

type dmaBuffer struct {
	// Leave room to align the usable scanline without relying on the target
	// compiler's stack alignment for byte arrays. Scaled presentation stores
	// all identical physical rows so they share one DMA transaction.
	data [DisplayWidth*6 + 3]byte
}

func pointerBits(pointer unsafe.Pointer) uintptr {
	return *(*uintptr)(unsafe.Pointer(&pointer))
}

func pointerBits32(pointer unsafe.Pointer) uint32 {
	return *(*uint32)(unsafe.Pointer(&pointer))
}

func dmaBufferData(buffer *dmaBuffer) []byte {
	data := buffer.data[:]
	offset := 0
	remainder := pointerBits32(unsafe.Pointer(&data[0])) & 3
	if remainder == 1 {
		offset = 3
	} else if remainder == 2 {
		offset = 2
	} else if remainder == 3 {
		offset = 1
	}
	return data[offset : offset+DisplayWidth*6]
}

func spiWait(mask uint32) {
	for load32(spi3Command)&mask != 0 {
	}
}

func spiInitialize() {
	store32(systemClock0, load32(systemClock0)|spi3Enable)
	store32(systemReset0, load32(systemReset0)|spi3Enable)
	store32(systemReset0, load32(systemReset0)&^spi3Enable)
	store32(spi3ClockGate, 7)
	store32(spi3Control, 0)
	// Match the official M5GFX StickS3 configuration: the 80 MHz peripheral
	// clock divided by two gives the ST7789 a 40 MHz mode-0 write clock.
	store32(spi3Clock, uint32((1<<12)|1))
	store32(spi3User, spi3UserMOSI)
	// Keep all hardware chip-select outputs disconnected. The panel's CS and
	// data/command lines remain ordinary GPIOs so command boundaries are clear.
	store32(spi3Misc, 0x3f)
	store32(spi3DMAConfig, 0)
	store32(spi3Command, spi3Update)
	spiWait(spi3Update)

	// Reserve GDMA channel 0 for SPI3 transmit. The StickS3 board package owns
	// this otherwise bare peripheral for the lifetime of the application.
	store32(systemClock1, load32(systemClock1)|gdmaEnable)
	store32(systemReset1, load32(systemReset1)|gdmaEnable)
	store32(systemReset1, load32(systemReset1)&^gdmaEnable)
	store32(gdmaMiscConfig, load32(gdmaMiscConfig)|(1<<4))
	store32(gdmaOutConfig1, load32(gdmaOutConfig1)|(1<<12))
	store32(gdmaOutPeripheral, 1) // SPI3

	configureGPIO(lcdMOSI, false)
	configureGPIO(lcdClock, false)
	connectGPIOOutput(lcdMOSI, 68)
	connectGPIOOutput(lcdClock, 66)
	enableGPIO(lcdMOSI, true)
	enableGPIO(lcdClock, true)
}

func spiDMAWait() bool {
	// Observe errors while the transfer is active: an underflow can prevent
	// SPI's USR bit from ever clearing. The SPI master completion contract does
	// not include GDMA TOTAL_EOF, so require the SPI transaction to finish and
	// the GDMA descriptor FSM to return to its documented parked state.
	for {
		status := load32(gdmaOutIntRaw)
		if status&(1<<2) != 0 || load32(spi3DMAIntRaw)&(1<<1) != 0 {
			return false
		}
		if load32(spi3Command)&spi3UserStart == 0 && load32(gdmaOutLink)&gdmaOutLinkParked != 0 {
			return true
		}
	}
}

func spiDMAStart(data []byte, descriptor *dmaDescriptor) {
	length := len(data)
	descriptor.control = uint32(length) | uint32(length)<<12 | 1<<30 | 1<<31
	descriptor.buffer = pointerBits(unsafe.Pointer(&data[0]))
	descriptor.next = 0

	// Reset the channel and SPI DMA FIFO before mounting the one-descriptor
	// transfer. GDMA link registers store the low 20 address bits for internal
	// DRAM descriptors on ESP32-S3.
	config := load32(gdmaOutConfig0)
	store32(gdmaOutConfig0, config|1)
	store32(gdmaOutConfig0, config&^1)
	store32(gdmaOutIntClear, 0xff)
	store32(spi3DMAConfig, load32(spi3DMAConfig)|1<<31)
	store32(spi3DMAConfig, load32(spi3DMAConfig)&^(1<<31))
	store32(spi3DMAIntClear, 1<<1)
	store32(spi3DMAConfig, load32(spi3DMAConfig)|spi3DMATX)
	store32(gdmaOutLink, uint32(pointerBits(unsafe.Pointer(descriptor)))&0xfffff|1<<21)
	store32(spi3DataLength, uint32(length*8-1))
	store32(spi3Command, spi3Update)
	spiWait(spi3Update)
	store32(spi3Command, spi3UserStart)
}

func spiStoreWord(index int, value uint32) {
	switch index {
	case 0:
		store32(spi3Data+0x00, value)
	case 1:
		store32(spi3Data+0x04, value)
	case 2:
		store32(spi3Data+0x08, value)
	case 3:
		store32(spi3Data+0x0c, value)
	case 4:
		store32(spi3Data+0x10, value)
	case 5:
		store32(spi3Data+0x14, value)
	case 6:
		store32(spi3Data+0x18, value)
	case 7:
		store32(spi3Data+0x1c, value)
	case 8:
		store32(spi3Data+0x20, value)
	case 9:
		store32(spi3Data+0x24, value)
	case 10:
		store32(spi3Data+0x28, value)
	case 11:
		store32(spi3Data+0x2c, value)
	case 12:
		store32(spi3Data+0x30, value)
	case 13:
		store32(spi3Data+0x34, value)
	case 14:
		store32(spi3Data+0x38, value)
	default:
		store32(spi3Data+0x3c, value)
	}
}

func spiWrite(data []byte) {
	for len(data) != 0 {
		count := len(data)
		if count > 64 {
			count = 64
		}
		for wordOffset := 0; wordOffset < count; wordOffset += 4 {
			word := uint32(0)
			wordLength := count - wordOffset
			if wordLength > 4 {
				wordLength = 4
			}
			for byteOffset := 0; byteOffset < wordLength; byteOffset++ {
				word |= uint32(data[wordOffset+byteOffset]) << uint(byteOffset*8)
			}
			spiStoreWord(wordOffset/4, word)
		}
		store32(spi3DataLength, uint32(count*8-1))
		store32(spi3Command, spi3Update)
		spiWait(spi3Update)
		store32(spi3Command, spi3UserStart)
		spiWait(spi3UserStart)
		data = data[count:]
	}
}

func spiWriteByte(value byte) {
	spiWrite([]byte{value})
}

func lcdCommand(command byte, data []byte) {
	setGPIO(lcdChipSelect, false)
	setGPIO(lcdDataCommand, false)
	spiWriteByte(command)
	setGPIO(lcdDataCommand, true)
	spiWrite(data)
	setGPIO(lcdChipSelect, true)
}

func lcdWindow(x0, y0, x1, y1 int) {
	x0 += 52
	x1 += 52
	y0 += 40
	y1 += 40
	lcdCommand(0x2a, []byte{byte(x0 >> 8), byte(x0), byte(x1 >> 8), byte(x1)})
	lcdCommand(0x2b, []byte{byte(y0 >> 8), byte(y0), byte(y1 >> 8), byte(y1)})
}

func lcdFillRectangle(x, y, width, height int, color uint16) {
	lcdWindow(x, y, x+width-1, y+height-1)
	setGPIO(lcdChipSelect, false)
	setGPIO(lcdDataCommand, false)
	spiWriteByte(0x2c)
	setGPIO(lcdDataCommand, true)
	pixels := width * height
	var buffer [64]byte
	for index := 0; index < len(buffer); index += 2 {
		buffer[index] = byte(color >> 8)
		buffer[index+1] = byte(color)
	}
	for pixels != 0 {
		count := pixels
		if count > len(buffer)/2 {
			count = len(buffer) / 2
		}
		spiWrite(buffer[:count*2])
		pixels -= count
	}
	setGPIO(lcdChipSelect, true)
}

func rgb565(red, green, blue byte) uint16 {
	return uint16(red&0xf8)<<8 | uint16(green&0xfc)<<3 | uint16(blue)>>3
}

// FillDisplay fills the complete panel without allocating a framebuffer.
func FillDisplay(red, green, blue byte) {
	lcdFillRectangle(0, 0, DisplayWidth, DisplayHeight, rgb565(red, green, blue))
}

func presentRGBA(pixels []byte, stride, scale, x0, y0, x1, y1 int) bool {
	width := DisplayWidth / scale
	height := DisplayHeight / scale
	if stride < width*4 || len(pixels) < stride*height {
		return false
	}
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > width {
		x1 = width
	}
	if y1 > height {
		y1 = height
	}
	if x0 >= x1 || y0 >= y1 {
		return true
	}

	lcdWindow(x0*scale, y0*scale, x1*scale-1, y1*scale-1)
	setGPIO(lcdChipSelect, false)
	setGPIO(lcdDataCommand, false)
	spiWriteByte(0x2c)
	setGPIO(lcdDataCommand, true)
	var buffer0 dmaBuffer
	var buffer1 dmaBuffer
	var descriptors [2]dmaDescriptor
	current := 0
	inFlight := false
	for y := y0; y < y1; y++ {
		var buffer []byte
		if current == 0 {
			buffer = dmaBufferData(&buffer0)
		} else {
			buffer = dmaBufferData(&buffer1)
		}
		used := 0
		for x := x0; x < x1; x++ {
			offset := y*stride + x*4
			color := rgb565(pixels[offset], pixels[offset+1], pixels[offset+2])
			for repeatX := 0; repeatX < scale; repeatX++ {
				buffer[used] = byte(color >> 8)
				buffer[used+1] = byte(color)
				used += 2
			}
		}
		rowBytes := used
		for repeatY := 1; repeatY < scale; repeatY++ {
			for index := 0; index < rowBytes; index++ {
				buffer[used] = buffer[index]
				used++
			}
		}
		if inFlight {
			if !spiDMAWait() {
				store32(spi3DMAConfig, load32(spi3DMAConfig)&^spi3DMATX)
				setGPIO(lcdChipSelect, true)
				return false
			}
		}
		spiDMAStart(buffer[:used], &descriptors[current])
		inFlight = true
		current = 1 - current
	}
	if inFlight {
		if !spiDMAWait() {
			store32(spi3DMAConfig, load32(spi3DMAConfig)&^spi3DMATX)
			setGPIO(lcdChipSelect, true)
			return false
		}
	}
	store32(spi3DMAConfig, load32(spi3DMAConfig)&^spi3DMATX)
	setGPIO(lcdChipSelect, true)
	return true
}

// PresentRGBA copies a rectangle from a full-screen RGBA8 buffer to the
// StickS3 LCD using double-buffered DMA scanlines and no second framebuffer.
func PresentRGBA(pixels []byte, stride, x0, y0, x1, y1 int) bool {
	return presentRGBA(pixels, stride, 1, x0, y0, x1, y1)
}

// PresentRGBA2x presents a 67x120 logical surface at two physical pixels per
// axis. The final LCD column remains black. This reduces a Forms framebuffer
// from 129600 bytes to 32160 bytes on memory-constrained systems.
func PresentRGBA2x(pixels []byte, stride, x0, y0, x1, y1 int) bool {
	return presentRGBA(pixels, stride, 2, x0, y0, x1, y1)
}

// PresentRGBA3x presents a 45x80 logical surface at three physical pixels per
// axis, using only 14400 bytes for a full Forms framebuffer.
func PresentRGBA3x(pixels []byte, stride, x0, y0, x1, y1 int) bool {
	return presentRGBA(pixels, stride, 3, x0, y0, x1, y1)
}

func presentSurface(surface *graphics.Surface, scale int) bool {
	width := DisplayWidth / scale
	height := DisplayHeight / scale
	if surface == nil || surface.Stride < width*4 || len(surface.Pixels) < surface.Stride*height {
		return false
	}
	dirty, ok := surface.DirtyRect()
	if !ok {
		return true
	}
	return presentRGBA(surface.Pixels, surface.Stride, scale,
		int(dirty.MinX), int(dirty.MinY), int(dirty.MaxX), int(dirty.MaxY))
}

// PresentSurface2x copies only the dirty part of a 67x120 Forms surface. The
// caller should reset the surface's dirty state after a successful transfer.
func PresentSurface2x(surface *graphics.Surface) bool {
	return presentSurface(surface, 2)
}

// PresentSurface3x copies only the dirty part of a 45x80 Forms surface. The
// caller should reset the surface's dirty state after a successful transfer.
func PresentSurface3x(surface *graphics.Surface) bool {
	return presentSurface(surface, 3)
}

func lcdInitialize() {
	lcdCommand(0xb7, []byte{0x35})
	lcdCommand(0xbb, []byte{0x28})
	lcdCommand(0xc0, []byte{0x0c})
	lcdCommand(0xc2, []byte{0x01, 0xff})
	lcdCommand(0xc3, []byte{0x10})
	lcdCommand(0xc4, []byte{0x20})
	lcdCommand(0xd0, []byte{0xa4, 0xa1})
	lcdCommand(0xb0, []byte{0x00, 0xc0})
	lcdCommand(0xe0, []byte{
		0xd0, 0x00, 0x02, 0x07, 0x0a, 0x28, 0x32,
		0x44, 0x42, 0x06, 0x0e, 0x12, 0x14, 0x17,
	})
	lcdCommand(0xe1, []byte{
		0xd0, 0x00, 0x02, 0x07, 0x0a, 0x28, 0x31,
		0x54, 0x47, 0x0e, 0x1c, 0x17, 0x1b, 0x1e,
	})
	lcdCommand(0x11, nil)
	Delay(300000)
	lcdCommand(0x38, nil)
	lcdCommand(0x3a, []byte{0x55})
	lcdCommand(0x36, []byte{0x08})
	lcdCommand(0x21, nil)
	lcdCommand(0x29, nil)
	Delay(100000)
}

func lcdDrawLines() {
	lcdFillRectangle(0, 0, DisplayWidth, DisplayHeight, lcdBlack)
	lcdFillRectangle(0, 0, DisplayWidth, 3, lcdWhite)
	lcdFillRectangle(0, 237, DisplayWidth, 3, lcdWhite)
	lcdFillRectangle(0, 0, 3, DisplayHeight, lcdWhite)
	lcdFillRectangle(132, 0, 3, DisplayHeight, lcdWhite)
	colors := []uint16{lcdRed, lcdYellow, lcdGreen, lcdCyan, lcdBlue, lcdMagenta}
	for index, color := range colors {
		lcdFillRectangle(8, 24+index*36, 119, 4, color)
	}
	for x := 16; x < 135; x += 17 {
		lcdFillRectangle(x, 8, 2, 224, lcdWhite)
	}
}

// InitializeDisplay powers and initializes the StickS3 ST7789. The backlight
// is enabled only after the panel is ready.
func InitializeDisplay() bool {
	if !EnableLCDPower() {
		return false
	}
	for _, pin := range []int{lcdChipSelect, lcdDataCommand, lcdReset} {
		configureGPIO(pin, false)
		setGPIO(pin, true)
		enableGPIO(pin, true)
	}
	spiInitialize()
	ConfigureBacklight()
	setGPIO(lcdReset, false)
	Delay(100000)
	setGPIO(lcdReset, true)
	Delay(300000)
	lcdCommand(0x01, nil)
	Delay(300000)
	lcdInitialize()
	lcdFillRectangle(0, 0, DisplayWidth, DisplayHeight, lcdBlack)
	SetBacklight(true)
	return true
}

// DrawButtonRectangle shows a red rectangle for Button A and a cyan rectangle
// for Button B. Releasing a button clears only its rectangle.
func DrawButtonRectangle(button int, pressed bool) {
	x := 15
	color := lcdRed
	if button == 1 {
		x = 75
		color = lcdCyan
	}
	if !pressed {
		color = lcdBlack
	}
	lcdFillRectangle(x, 85, 45, 2, color)
	lcdFillRectangle(x, 153, 45, 2, color)
	lcdFillRectangle(x, 87, 2, 66, color)
	lcdFillRectangle(x+43, 87, 2, 66, color)
}

// DrawLineDiagnostic enables only the LCD rail, initializes the StickS3
// ST7789, and draws a sparse line test. The backlight stays off until all panel
// writes are complete.
func DrawLineDiagnostic() bool {
	if !InitializeDisplay() {
		return false
	}
	lcdDrawLines()
	return true
}

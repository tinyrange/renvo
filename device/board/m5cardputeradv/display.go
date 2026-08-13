package board

import (
	"unsafe"

	"renvo.dev/device/esp32s3"
	"renvo.dev/device/gpio"
	"renvo.dev/device/mmio"
	"renvo.dev/std/graphics"
)

const (
	displayWidth  = 240
	displayHeight = 135
)

const (
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
	spi3DMAEnable     = uint32(1 << 27)
	gdmaEnable        = uint32(1 << 6)
	spi3Update        = uint32(1 << 23)
	spi3UserStart     = uint32(1 << 24)
	spi3UserMOSI      = uint32(1 << 27)
	spi3DMATX         = uint32(1 << 28)
	gdmaOutLinkParked = uint32(1 << 23)
)

var (
	lcdMOSI        = esp32s3.GPIO(35)
	lcdClock       = esp32s3.GPIO(36)
	lcdChipSelect  = esp32s3.GPIO(37)
	lcdDataCommand = esp32s3.GPIO(34)
	lcdReset       = esp32s3.GPIO(33)
	lcdBacklight   = esp32s3.GPIO(38)
)

// ESP32-S3 GDMA descriptors are exactly three contiguous 32-bit words. Keep
// this as an array rather than a Go struct: Renvo's current 32-bit target
// representation gives struct fields widened storage slots.
type dmaDescriptor [3]uint32

// Store scanlines as words so every GDMA buffer is aligned by construction.
// The maximum scaled row is displayWidth*6 bytes; round its storage up to a
// complete word while passing the exact byte count to SPI.
type dmaBuffer [(displayWidth*6 + 3) / 4]uint32

func pointerBits(pointer unsafe.Pointer) uintptr {
	return *(*uintptr)(unsafe.Pointer(&pointer))
}

func pointerBits32(pointer unsafe.Pointer) uint32 {
	return *(*uint32)(unsafe.Pointer(&pointer))
}

func dmaStoreByte(buffer *dmaBuffer, offset int, value byte) {
	words := buffer[:]
	wordIndex := offset / 4
	shift := uint(offset%4) * 8
	mask := uint32(0xff) << shift
	words[wordIndex] = words[wordIndex]&^mask | uint32(value)<<shift
}

func dmaLoadByte(buffer *dmaBuffer, offset int) byte {
	words := buffer[:]
	wordIndex := offset / 4
	shift := uint(offset%4) * 8
	return byte(words[wordIndex] >> shift)
}

func spiWait(mask uint32) {
	for mmio.Load32(spi3Command)&mask != 0 {
	}
}

func spiInitialize() {
	mmio.Store32(systemClock0, mmio.Load32(systemClock0)|spi3Enable|spi3DMAEnable)
	mmio.Store32(systemReset0, mmio.Load32(systemReset0)|spi3Enable|spi3DMAEnable)
	mmio.Store32(systemReset0, mmio.Load32(systemReset0)&^(spi3Enable|spi3DMAEnable))
	mmio.Store32(spi3ClockGate, 7)
	mmio.Store32(spi3Control, 0)
	// Match the official M5GFX StickS3 configuration: the 80 MHz peripheral
	// clock divided by two gives the ST7789 a 40 MHz mode-0 write clock.
	mmio.Store32(spi3Clock, uint32((1<<12)|1))
	mmio.Store32(spi3User, spi3UserMOSI)
	// Keep all hardware chip-select outputs disconnected. The panel's CS and
	// data/command lines remain ordinary GPIOs so command boundaries are clear.
	mmio.Store32(spi3Misc, 0x3f)
	mmio.Store32(spi3DMAConfig, 0)
	mmio.Store32(spi3Command, spi3Update)
	spiWait(spi3Update)

	// Reserve GDMA channel 0 for SPI3 transmit. The Cardputer Adv board package owns
	// this otherwise bare peripheral for the lifetime of the application.
	mmio.Store32(systemClock1, mmio.Load32(systemClock1)|gdmaEnable)
	mmio.Store32(systemReset1, mmio.Load32(systemReset1)|gdmaEnable)
	mmio.Store32(systemReset1, mmio.Load32(systemReset1)&^gdmaEnable)
	mmio.Store32(gdmaMiscConfig, mmio.Load32(gdmaMiscConfig)|(1<<4))
	mmio.Store32(gdmaOutConfig1, mmio.Load32(gdmaOutConfig1)|(1<<12))
	mmio.Store32(gdmaOutPeripheral, 1) // SPI3

	_ = lcdMOSI.ConfigureOutputSignal(68)
	_ = lcdClock.ConfigureOutputSignal(66)
}

func spiDMAWait() bool {
	// Observe errors while the transfer is active: an underflow can prevent
	// SPI's USR bit from ever clearing. The SPI master completion contract does
	// not include GDMA TOTAL_EOF, so require the SPI transaction to finish and
	// the GDMA descriptor FSM to return to its documented parked state.
	for {
		status := mmio.Load32(gdmaOutIntRaw)
		if status&(1<<2) != 0 || mmio.Load32(spi3DMAIntRaw)&(1<<1) != 0 {
			return false
		}
		if mmio.Load32(spi3Command)&spi3UserStart == 0 && mmio.Load32(gdmaOutLink)&gdmaOutLinkParked != 0 {
			return true
		}
	}
}

func spiDMAStart(buffer *dmaBuffer, length int, descriptor *dmaDescriptor) {
	descriptor[0] = uint32(length) | uint32(length)<<12 | 1<<30 | 1<<31
	descriptor[1] = pointerBits32(unsafe.Pointer(&buffer[0]))
	descriptor[2] = 0

	// Reset the channel and SPI DMA FIFO before mounting the one-descriptor
	// transfer. GDMA link registers store the low 20 address bits for internal
	// DRAM descriptors on ESP32-S3.
	config := mmio.Load32(gdmaOutConfig0)
	mmio.Store32(gdmaOutConfig0, config|1)
	mmio.Store32(gdmaOutConfig0, config&^1)
	mmio.Store32(gdmaOutIntClear, 0xff)
	mmio.Store32(spi3DMAConfig, mmio.Load32(spi3DMAConfig)|1<<31)
	mmio.Store32(spi3DMAConfig, mmio.Load32(spi3DMAConfig)&^(1<<31))
	mmio.Store32(spi3DMAIntClear, 1<<1)
	mmio.Store32(spi3DMAConfig, mmio.Load32(spi3DMAConfig)|spi3DMATX)
	mmio.Store32(spi3DataLength, uint32(length*8-1))
	descriptorAddress := uint32(pointerBits(unsafe.Pointer(descriptor))) & 0xfffff
	mmio.Store32(gdmaOutLink, descriptorAddress)
	mmio.Store32(gdmaOutLink, descriptorAddress|1<<21)
	mmio.Store32(spi3Command, spi3Update)
	spiWait(spi3Update)
	mmio.Store32(spi3Command, spi3UserStart)
}

func spiStoreWord(index int, value uint32) {
	switch index {
	case 0:
		mmio.Store32(spi3Data+0x00, value)
	case 1:
		mmio.Store32(spi3Data+0x04, value)
	case 2:
		mmio.Store32(spi3Data+0x08, value)
	case 3:
		mmio.Store32(spi3Data+0x0c, value)
	case 4:
		mmio.Store32(spi3Data+0x10, value)
	case 5:
		mmio.Store32(spi3Data+0x14, value)
	case 6:
		mmio.Store32(spi3Data+0x18, value)
	case 7:
		mmio.Store32(spi3Data+0x1c, value)
	case 8:
		mmio.Store32(spi3Data+0x20, value)
	case 9:
		mmio.Store32(spi3Data+0x24, value)
	case 10:
		mmio.Store32(spi3Data+0x28, value)
	case 11:
		mmio.Store32(spi3Data+0x2c, value)
	case 12:
		mmio.Store32(spi3Data+0x30, value)
	case 13:
		mmio.Store32(spi3Data+0x34, value)
	case 14:
		mmio.Store32(spi3Data+0x38, value)
	default:
		mmio.Store32(spi3Data+0x3c, value)
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
		mmio.Store32(spi3DataLength, uint32(count*8-1))
		mmio.Store32(spi3Command, spi3Update)
		spiWait(spi3Update)
		mmio.Store32(spi3Command, spi3UserStart)
		spiWait(spi3UserStart)
		data = data[count:]
	}
}

func spiWriteByte(value byte) {
	spiWrite([]byte{value})
}

func lcdCommand(command byte, data []byte) {
	lcdChipSelect.Set(false)
	lcdDataCommand.Set(false)
	spiWriteByte(command)
	lcdDataCommand.Set(true)
	spiWrite(data)
	lcdChipSelect.Set(true)
}

func lcdWindow(x0, y0, x1, y1 int) {
	// M5GFX's ST7789 rotation 1 swaps the configured 52/40 panel offsets.
	// The reversed row axis contributes the one-pixel asymmetry: 240-(135+52).
	x0 += 40
	x1 += 40
	y0 += 53
	y1 += 53
	lcdCommand(0x2a, []byte{byte(x0 >> 8), byte(x0), byte(x1 >> 8), byte(x1)})
	lcdCommand(0x2b, []byte{byte(y0 >> 8), byte(y0), byte(y1 >> 8), byte(y1)})
}

func lcdFillRectangle(x, y, width, height int, color uint16) {
	lcdWindow(x, y, x+width-1, y+height-1)
	lcdChipSelect.Set(false)
	lcdDataCommand.Set(false)
	spiWriteByte(0x2c)
	lcdDataCommand.Set(true)
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
	lcdChipSelect.Set(true)
}

func rgb565(red, green, blue byte) uint16 {
	return uint16(red&0xf8)<<8 | uint16(green&0xfc)<<3 | uint16(blue)>>3
}

// fillDisplay fills the complete panel without allocating a framebuffer.
func fillDisplay(red, green, blue byte) {
	lcdFillRectangle(0, 0, displayWidth, displayHeight, rgb565(red, green, blue))
}

func presentRGBAScaled(pixels []byte, stride, scale, x0, y0, x1, y1 int) bool {
	width := displayWidth / scale
	height := displayHeight / scale
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
	lcdChipSelect.Set(false)
	lcdDataCommand.Set(false)
	spiWriteByte(0x2c)
	lcdDataCommand.Set(true)
	var buffer0 dmaBuffer
	var buffer1 dmaBuffer
	var descriptors [2]dmaDescriptor
	current := 0
	inFlight := false
	for y := y0; y < y1; y++ {
		var buffer *dmaBuffer
		if current == 0 {
			buffer = &buffer0
		} else {
			buffer = &buffer1
		}
		used := 0
		if scale == 2 {
			// Two identical RGB565 pixels are exactly one DMA word. Packing the
			// common Forms path directly avoids per-byte read/modify/write calls.
			words := buffer[:]
			for x := x0; x < x1; x++ {
				offset := y*stride + x*4
				color := rgb565(pixels[offset], pixels[offset+1], pixels[offset+2])
				high := uint32(byte(color >> 8))
				low := uint32(byte(color))
				words[used/4] = high | low<<8 | high<<16 | low<<24
				used += 4
			}
			rowWords := used / 4
			for index := 0; index < rowWords; index++ {
				words[rowWords+index] = words[index]
			}
			used *= 2
		} else {
			for x := x0; x < x1; x++ {
				offset := y*stride + x*4
				color := rgb565(pixels[offset], pixels[offset+1], pixels[offset+2])
				for repeatX := 0; repeatX < scale; repeatX++ {
					dmaStoreByte(buffer, used, byte(color>>8))
					dmaStoreByte(buffer, used+1, byte(color))
					used += 2
				}
			}
			rowBytes := used
			for repeatY := 1; repeatY < scale; repeatY++ {
				for index := 0; index < rowBytes; index++ {
					dmaStoreByte(buffer, used, dmaLoadByte(buffer, index))
					used++
				}
			}
		}
		if inFlight {
			if !spiDMAWait() {
				mmio.Store32(spi3DMAConfig, mmio.Load32(spi3DMAConfig)&^spi3DMATX)
				lcdChipSelect.Set(true)
				return false
			}
		}
		spiDMAStart(buffer, used, &descriptors[current])
		inFlight = true
		current = 1 - current
	}
	if inFlight {
		if !spiDMAWait() {
			mmio.Store32(spi3DMAConfig, mmio.Load32(spi3DMAConfig)&^spi3DMATX)
			lcdChipSelect.Set(true)
			return false
		}
	}
	mmio.Store32(spi3DMAConfig, mmio.Load32(spi3DMAConfig)&^spi3DMATX)
	lcdChipSelect.Set(true)
	return true
}

// presentRGBA copies a rectangle from a full-screen RGBA8 buffer to the
// Cardputer Adv LCD using double-buffered DMA scanlines and no second framebuffer.
func presentRGBA(pixels []byte, stride, x0, y0, x1, y1 int) bool {
	return presentRGBAScaled(pixels, stride, 1, x0, y0, x1, y1)
}

// presentRGBA2x presents a 67x120 logical surface at two physical pixels per
// axis. The final LCD column remains black. This reduces a Forms framebuffer
// from 129600 bytes to 32160 bytes on memory-constrained systems.
func presentRGBA2x(pixels []byte, stride, x0, y0, x1, y1 int) bool {
	return presentRGBAScaled(pixels, stride, 2, x0, y0, x1, y1)
}

// presentRGBA3x presents a 45x80 logical surface at three physical pixels per
// axis, using only 14400 bytes for a full Forms framebuffer.
func presentRGBA3x(pixels []byte, stride, x0, y0, x1, y1 int) bool {
	return presentRGBAScaled(pixels, stride, 3, x0, y0, x1, y1)
}

func presentSurface(surface *graphics.Surface, scale int) bool {
	width := displayWidth / scale
	height := displayHeight / scale
	if surface == nil || surface.Stride < width*4 || len(surface.Pixels) < surface.Stride*height {
		return false
	}
	dirty, ok := surface.DirtyRect()
	if !ok {
		return true
	}
	return presentRGBAScaled(surface.Pixels, surface.Stride, scale,
		int(dirty.MinX), int(dirty.MinY), int(dirty.MaxX), int(dirty.MaxY))
}

// presentSurface2x copies only the dirty part of a 120x67 Forms surface. The
// caller should reset the surface's dirty state after a successful transfer.
func presentSurface2x(surface *graphics.Surface) bool {
	return presentSurface(surface, 2)
}

// presentSurface3x copies only the dirty part of an 80x45 Forms surface. The
// caller should reset the surface's dirty state after a successful transfer.
func presentSurface3x(surface *graphics.Surface) bool {
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
	Clock.DelayMilliseconds(120)
	lcdCommand(0x38, nil)
	lcdCommand(0x3a, []byte{0x55})
	lcdCommand(0x36, []byte{0x6c})
	lcdCommand(0x21, nil)
	lcdCommand(0x29, nil)
	Clock.DelayMilliseconds(20)
}

func lcdDrawLines() {
	lcdFillRectangle(0, 0, displayWidth, displayHeight, lcdBlack)
	lcdFillRectangle(0, 0, displayWidth, 3, lcdWhite)
	lcdFillRectangle(0, displayHeight-3, displayWidth, 3, lcdWhite)
	lcdFillRectangle(0, 0, 3, displayHeight, lcdWhite)
	lcdFillRectangle(displayWidth-3, 0, 3, displayHeight, lcdWhite)
	colors := []uint16{lcdRed, lcdYellow, lcdGreen, lcdCyan, lcdBlue, lcdMagenta}
	for index, color := range colors {
		lcdFillRectangle(20+index*36, 12, 4, displayHeight-24, color)
	}
	for y := 20; y < displayHeight; y += 19 {
		lcdFillRectangle(8, y, displayWidth-16, 2, lcdWhite)
	}
}

// initializeDisplay resets and initializes the Cardputer Adv ST7789V2. GPIO38
// supplies both the LCD backlight and RGB LED, so it is enabled only after the
// panel is ready.
func initializeDisplay() bool {
	for _, pin := range []*esp32s3.Pin{lcdChipSelect, lcdDataCommand, lcdReset} {
		pin.Set(true)
		_ = pin.Configure(gpio.Config{Direction: gpio.Output})
	}
	spiInitialize()
	lcdBacklight.Set(false)
	_ = lcdBacklight.Configure(gpio.Config{Direction: gpio.Output})
	lcdReset.Set(false)
	Clock.DelayMilliseconds(10)
	lcdReset.Set(true)
	Clock.DelayMilliseconds(120)
	lcdCommand(0x01, nil)
	Clock.DelayMilliseconds(120)
	lcdInitialize()
	lcdFillRectangle(0, 0, displayWidth, displayHeight, lcdBlack)
	lcdBacklight.Set(true)
	return true
}

func drawKey(row, column int, pressed bool) bool {
	if row < 0 || row >= 4 || column < 0 || column >= 14 {
		return false
	}
	x := 9 + column*16
	y := 52 + row*19
	color := lcdGreen
	if !pressed {
		color = lcdMagenta
	}
	lcdFillRectangle(x, y, 12, 13, color)
	return true
}

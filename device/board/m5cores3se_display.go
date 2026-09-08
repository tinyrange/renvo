//go:build m5cores3se

package board

import (
	"unsafe"

	"renvo.dev/device/esp32s3"
	"renvo.dev/device/gpio"
	"renvo.dev/device/mmio"
	"renvo.dev/std/graphics"
)

const (
	displayWidth  = 320
	displayHeight = 240
)

const (
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
	lcdMOSI        = esp32s3.GPIO(37)
	lcdClock       = esp32s3.GPIO(36)
	lcdChipSelect  = esp32s3.GPIO(3)
	lcdDataCommand = esp32s3.GPIO(35)
)

// ESP32-S3 GDMA descriptors are exactly three contiguous 32-bit words. Keep
// this as an array rather than a Go struct: Renvo's current 32-bit target
// representation gives struct fields widened storage slots.
type dmaDescriptor [3]uint32

// Store scanlines as words so every GDMA buffer is aligned by construction.
// Each transfer contains two physical RGB565 rows.
type dmaBuffer [displayWidth]uint32

func pointerBits(pointer unsafe.Pointer) uintptr {
	return *(*uintptr)(unsafe.Pointer(&pointer))
}

func pointerBits32(pointer unsafe.Pointer) uint32 {
	return *(*uint32)(unsafe.Pointer(&pointer))
}

// The surface is already RGB565. Duplicate each pixel for the 2x scale and
// order its two bytes for SPI; no RGB color conversion occurs at presentation.
func packRGB5652xRow(pixels []byte, stride, x0, x1, y int, buffer *dmaBuffer) int {
	words := buffer[:]
	count := x1 - x0
	offset := y*stride + x0*2
	for i := 0; i < count; i++ {
		pair := uint32(pixels[offset+1]) | uint32(pixels[offset])<<8
		pair |= pair << 16
		words[i] = pair
		words[count+i] = pair
		offset += 2
	}
	return count * 8
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
	// Match the official M5GFX CoreS3-SE configuration: the 80 MHz peripheral
	// clock divided by two gives the ILI9342C a 40 MHz mode-0 write clock.
	mmio.Store32(spi3Clock, uint32((1<<12)|1))
	mmio.Store32(spi3User, spi3UserMOSI)
	// Keep all hardware chip-select outputs disconnected. The panel's CS and
	// data/command lines remain ordinary GPIOs so command boundaries are clear.
	mmio.Store32(spi3Misc, 0x3f)
	mmio.Store32(spi3DMAConfig, 0)
	mmio.Store32(spi3Command, spi3Update)
	spiWait(spi3Update)

	// Reserve GDMA channel 0 for SPI3 transmit. The CoreS3-SE board package owns
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

func presentPixels2x(pixels []byte, stride, x0, y0, x1, y1 int) bool {
	const scale = 2
	width := displayWidth / scale
	height := displayHeight / scale
	if stride < width*2 || len(pixels) < stride*height {
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
		used := packRGB5652xRow(pixels, stride, x0, x1, y, buffer)
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

func presentSurface2x(surface *graphics.Surface) bool {
	const scale = 2
	width := displayWidth / scale
	height := displayHeight / scale
	if surface == nil || surface.Format != graphics.PixelRGB565 || surface.Width != width || surface.Height != height || surface.Stride < width*2 || len(surface.Pixels) < surface.Stride*height {
		return false
	}
	dirty, ok := surface.DirtyRect()
	if !ok {
		return true
	}
	return presentPixels2x(surface.Pixels, surface.Stride,
		int(dirty.MinX), int(dirty.MinY), int(dirty.MaxX), int(dirty.MaxY))
}

// Initialization register values follow M5GFX's Panel_ILI9342.
// https://github.com/m5stack/M5GFX/blob/master/src/lgfx/v1/panel/Panel_ILI9342.hpp
func lcdInitialize() {
	lcdCommand(0xc8, []byte{0xff, 0x93, 0x42})
	lcdCommand(0xc0, []byte{0x12, 0x12})
	lcdCommand(0xc1, []byte{0x03})
	lcdCommand(0xc5, []byte{0xf2})
	lcdCommand(0xb0, []byte{0xe0})
	lcdCommand(0xf6, []byte{1, 0, 0})
	lcdCommand(0xe0, []byte{0, 0x0c, 0x11, 4, 0x11, 8, 0x37, 0x89, 0x4c, 6, 0x0c, 0x0a, 0x2e, 0x34, 0x0f})
	lcdCommand(0xe1, []byte{0, 0x0b, 0x11, 5, 0x13, 9, 0x33, 0x67, 0x48, 7, 0x0e, 0x0b, 0x2e, 0x33, 0x0f})
	lcdCommand(0xb6, []byte{8, 0x82, 0x1d, 4})
	lcdCommand(0x3a, []byte{0x55})
	lcdCommand(0x36, []byte{0x08})
	lcdCommand(0x21, nil)
	lcdCommand(0x38, nil)
	lcdCommand(0x11, nil)
	Clock.DelayMilliseconds(120)
	lcdCommand(0x29, nil)
	Clock.DelayMilliseconds(20)
}

func initializeDisplay() bool {
	if !initializePower() {
		return false
	}
	// The SD slot shares SPI; keep its chip select inactive.
	sdCS := esp32s3.GPIO(4)
	for _, pin := range []*esp32s3.Pin{lcdChipSelect, lcdDataCommand, sdCS} {
		pin.Set(true)
		_ = pin.Configure(gpio.Config{Direction: gpio.Output})
	}
	spiInitialize()
	if !coreUpdate(0x58, 3, 2, 0) {
		return false
	}
	Clock.DelayMilliseconds(10)
	if !coreUpdate(0x58, 3, 2, 2) {
		return false
	}
	Clock.DelayMilliseconds(120)
	lcdCommand(1, nil)
	Clock.DelayMilliseconds(120)
	lcdInitialize()
	fillDisplay(0, 0, 0)
	return coreWrite(0x34, 0x99, 24) && coreUpdate(0x34, 0x90, 0x80, 0x80)
}

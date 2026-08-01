package board

const (
	lcdMOSI        = 39
	lcdClock       = 40
	lcdChipSelect  = 41
	lcdDataCommand = 45
	lcdReset       = 21
	lcdBlack       = uint16(0x0000)
	lcdWhite       = uint16(0xffff)
	lcdRed         = uint16(0xf800)
	lcdYellow      = uint16(0xffe0)
	lcdGreen       = uint16(0x07e0)
	lcdCyan        = uint16(0x07ff)
	lcdBlue        = uint16(0x001f)
	lcdMagenta     = uint16(0xf81f)
	spi3Base       = uintptr(0x60025000)
	spi3Command    = spi3Base + 0x00
	spi3Control    = spi3Base + 0x08
	spi3Clock      = spi3Base + 0x0c
	spi3User       = spi3Base + 0x10
	spi3DataLength = spi3Base + 0x1c
	spi3Misc       = spi3Base + 0x20
	spi3DMAConfig  = spi3Base + 0x30
	spi3Data       = spi3Base + 0x98
	spi3ClockGate  = spi3Base + 0xe8
	systemClock0   = uintptr(0x600c0018)
	systemReset0   = uintptr(0x600c0020)
	spi3Enable     = uint32(1 << 16)
	spi3Update     = uint32(1 << 23)
	spi3UserStart  = uint32(1 << 24)
	spi3UserMOSI   = uint32(1 << 27)
)

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
	// 80 MHz peripheral clock / 8 gives a deliberately conservative 10 MHz
	// mode-0 LCD clock: pre=1, n=8, high=4, low=8.
	store32(spi3Clock, uint32((7<<12)|(3<<6)|7))
	store32(spi3User, spi3UserMOSI)
	// Keep all hardware chip-select outputs disconnected. The panel's CS and
	// data/command lines remain ordinary GPIOs so command boundaries are clear.
	store32(spi3Misc, 0x3f)
	store32(spi3DMAConfig, 0)
	store32(spi3Command, spi3Update)
	spiWait(spi3Update)

	configureGPIO(lcdMOSI, false)
	configureGPIO(lcdClock, false)
	connectGPIOOutput(lcdMOSI, 68)
	connectGPIOOutput(lcdClock, 66)
	enableGPIO(lcdMOSI, true)
	enableGPIO(lcdClock, true)
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
	lcdFillRectangle(0, 0, 135, 240, lcdBlack)
	lcdFillRectangle(0, 0, 135, 3, lcdWhite)
	lcdFillRectangle(0, 237, 135, 3, lcdWhite)
	lcdFillRectangle(0, 0, 3, 240, lcdWhite)
	lcdFillRectangle(132, 0, 3, 240, lcdWhite)
	colors := []uint16{lcdRed, lcdYellow, lcdGreen, lcdCyan, lcdBlue, lcdMagenta}
	for index, color := range colors {
		lcdFillRectangle(8, 24+index*36, 119, 4, color)
	}
	for x := 16; x < 135; x += 17 {
		lcdFillRectangle(x, 8, 2, 224, lcdWhite)
	}
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
	lcdDrawLines()
	SetBacklight(true)
	return true
}

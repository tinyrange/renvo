package board

type panelTransport struct{}

func waitPanelFIFO(mask uint32) {
	for load32(dsiBase+0x74)&mask != 0 {
	}
}

func (panelTransport) Command(command byte, data []byte) {
	size := len(data) + 1
	first := uint32(command)
	merged := len(data)
	if merged > 3 {
		merged = 3
	}
	for index := 0; index < merged; index++ {
		first |= uint32(data[index]) << uint((index+1)*8)
	}

	if size > 2 {
		waitPanelFIFO(1 << 3)
		store32(dsiBase+0x70, first)
		for offset := merged; offset < len(data); offset += 4 {
			word := uint32(0)
			remaining := len(data) - offset
			if remaining > 4 {
				remaining = 4
			}
			for index := 0; index < remaining; index++ {
				word |= uint32(data[offset+index]) << uint(index*8)
			}
			waitPanelFIFO(1 << 3)
			store32(dsiBase+0x70, word)
		}
		waitPanelFIFO(1 << 1)
		store32(dsiBase+0x6c, uint32(size)<<8|0x39)
		return
	}

	dataType := uint32(0x05)
	if size == 2 {
		dataType = 0x15
	}
	waitPanelFIFO(1 << 1)
	store32(dsiBase+0x6c, first<<8|dataType)
}

func (panelTransport) Delay(milliseconds int) {
	panelDelay(milliseconds)
}

func panelDelay(milliseconds int) {
	// The target runs at 360 MHz. This conservative software delay is used
	// only during controller initialization, where longer waits are harmless.
	delay(milliseconds * 15000)
}

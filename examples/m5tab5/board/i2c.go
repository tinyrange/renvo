package board

const (
	touchSDA = 31
	touchSCL = 32

	i2cBase      = uintptr(0x500c4000)
	i2cClockGate = clockBase + 0x1c
	i2cClock     = clockBase + 0x40
	i2cReset     = clockBase + 0xc4

	i2cControl    = i2cBase + 0x04
	i2cFIFOConfig = i2cBase + 0x18
	i2cData       = i2cBase + 0x1c
	i2cInterrupt  = i2cBase + 0x20
	i2cClear      = i2cBase + 0x24
	i2cCommands   = i2cBase + 0x58

	i2cOperationRestart = uint32(6 << 11)
	i2cOperationWrite   = uint32(1 << 11)
	i2cOperationStop    = uint32(2 << 11)
	i2cOperationRead    = uint32(3 << 11)
	i2cOperationEnd     = uint32(4 << 11)
)

var lastI2CFailure int
var lastStartFailure int
var lastI2CEvent uint32
var lastI2CByte uint32
var lastI2CState uint32
var lastI2CFIFO uint32

func i2cCommand(index int, value uint32) {
	store32(i2cCommands+uintptr(index*4), value)
}

func i2cResetFIFOs() {
	value := load32(i2cFIFOConfig)
	store32(i2cFIFOConfig, value|3<<12)
	store32(i2cFIFOConfig, value&^(3<<12))
}

func i2cRecover() {
	// NACK aborts the command list before STOP. Reset the master FSM so the
	// following transaction begins from IDLE, as ESP-IDF's driver does.
	update32(i2cControl, 0, 1<<10)
	store32(i2cClear, 0x7ffff)
}

func i2cWaitIdle() bool {
	for attempt := 0; attempt < 10000; attempt++ {
		if load32(i2cBase+0x08)&(1<<4) == 0 {
			return true
		}
		delay(100)
	}
	lastI2CEvent = 0xffffffff
	lastI2CState = load32(i2cBase + 0x08)
	lastI2CFIFO = load32(i2cBase + 0x14)
	i2cRecover()
	return false
}

func i2cBegin(lastCommand int) bool {
	return i2cBeginWithEvents(lastCommand)
}

func i2cBeginWithEvents(lastCommand int) bool {
	// Clear every event from the preceding transaction, synchronize the shadow
	// configuration registers, then start the command list.
	store32(i2cClear, 0x7ffff)
	lastI2CEvent = 0
	lastI2CByte = 0
	lastI2CState = 0
	lastI2CFIFO = 0
	update32(i2cControl, 0, 1<<11)
	update32(i2cControl, 0, 1<<5)
	operation := load32(i2cCommands+uintptr(lastCommand*4)) >> 11 & 7
	completion := uint32(0)
	if operation == 2 {
		completion = 1 << 7
	} else if operation == 4 {
		completion = 1 << 3
	}
	for attempt := 0; attempt < 10000; attempt++ {
		events := load32(i2cInterrupt)
		if completion != 0 && events&completion != 0 ||
			completion == 0 && load32(i2cCommands+uintptr(lastCommand*4))&(1<<31) != 0 {
			if events&((1<<5)|(1<<8)|(1<<10)|(1<<13)) != 0 {
				lastI2CEvent = load32(i2cInterrupt)
				lastI2CByte = load32(i2cBase+0x14) >> 10 & 0x1f
				i2cRecover()
				return false
			}
			// STOP_DETECT can precede BUS_BUSY deassertion. Do not let a rapid
			// touch poll reset the FIFOs while the hardware still owns the bus.
			if operation == 2 {
				return i2cWaitIdle()
			}
			return true
		}
		errorMask := uint32((1 << 5) | (1 << 8) | (1 << 10) | (1 << 13))
		if load32(i2cInterrupt)&errorMask != 0 {
			lastI2CEvent = load32(i2cInterrupt)
			lastI2CByte = load32(i2cBase+0x14) >> 10 & 0x1f
			i2cRecover()
			return false
		}
		// Give the 100 kHz command engine time to advance without burning the
		// entire timeout in a tight CPU-speed loop.
		delay(100)
	}
	lastI2CEvent = 0xffffffff
	lastI2CState = load32(i2cBase + 0x08)
	lastI2CFIFO = load32(i2cBase + 0x14)
	i2cRecover()
	return false
}

func i2cContinueRead(data []byte) bool {
	command := 0
	i2cCommand(command, i2cOperationRestart)
	command++
	i2cCommand(command, i2cOperationWrite|1<<8|1)
	command++
	if len(data) > 1 {
		i2cCommand(command, i2cOperationRead|uint32(len(data)-1))
		command++
	}
	i2cCommand(command, i2cOperationRead|1<<10|1)
	command++
	i2cCommand(command, i2cOperationStop)
	command++
	i2cCommand(command, i2cOperationEnd)

	store32(i2cClear, 0x7ffff)
	lastI2CEvent = 0
	lastI2CByte = 0
	lastI2CState = 0
	lastI2CFIFO = 0
	update32(i2cControl, 0, 1<<11)
	update32(i2cControl, 0, 1<<5)
	offset := 0
	errorMask := uint32((1 << 2) | (1 << 5) | (1 << 8) | (1 << 10) | (1 << 12) | (1 << 13))
	for attempt := 0; attempt < 200000; attempt++ {
		// Drain while the transaction is active. A single uninterrupted read is
		// important for ST7121 coordinate snapshots, but its 70-byte report is
		// larger than the P4's 32-byte RX FIFO.
		available := int(load32(i2cBase+0x08) >> 8 & 0x3f)
		for available > 0 && offset < len(data) {
			data[offset] = byte(load32(i2cData))
			offset++
			available--
		}
		events := load32(i2cInterrupt)
		if events&errorMask != 0 {
			lastI2CEvent = events
			lastI2CByte = load32(i2cBase+0x14) >> 10 & 0x1f
			i2cRecover()
			return false
		}
		if events&(1<<7) != 0 {
			available = int(load32(i2cBase+0x08) >> 8 & 0x3f)
			for available > 0 && offset < len(data) {
				data[offset] = byte(load32(i2cData))
				offset++
				available--
			}
			if offset != len(data) {
				lastI2CEvent = 0xffffffff
				lastI2CState = load32(i2cBase + 0x08)
				lastI2CFIFO = load32(i2cBase + 0x14)
				i2cRecover()
				return false
			}
			return i2cWaitIdle()
		}
		delay(20)
	}
	lastI2CEvent = 0xffffffff
	lastI2CState = load32(i2cBase + 0x08)
	lastI2CFIFO = load32(i2cBase + 0x14)
	i2cRecover()
	return false
}

func configureI2C(dataPin, clockPin int) {
	// Enable and reset I2C0 through the ESP32-P4 HP clock controller.
	update32(i2cClockGate, 0, 1<<12)
	update32(i2cReset, 0, 1<<22)
	update32(i2cReset, 1<<22, 0)

	// Route the peripheral's open-drain outputs and inputs through the GPIO
	// matrix. These values match the live factory BSP configuration:
	// signal 69 is SDA on GPIO31 and signal 68 is SCL on GPIO32.
	configureGPIO(dataPin, true, true)
	configureGPIO(clockPin, true, true)
	// configureGPIO preserves the reset drive-strength field.  The factory BSP
	// clears that field for this open-drain bus; use its measured IOMUX value
	// exactly on both internal I2C pins.
	store32(ioMuxBase+4+uintptr(dataPin*4), 0x1300)
	store32(ioMuxBase+4+uintptr(clockPin*4), 0x1300)
	setGPIO(dataPin, true)
	setGPIO(clockPin, true)
	update32(gpioBase+0xf0, 0, 1<<2)
	update32(gpioBase+0xf4, 0, 1<<2)
	store32(gpioBase+0x5d4, 69)
	store32(gpioBase+0x5d8, 68)
	store32(gpioBase+0x268, 1<<7|uint32(clockPin))
	store32(gpioBase+0x26c, 1<<7|uint32(dataPin))

	// XTAL source, controller enabled, divide by one. The timing values are the
	// 100 kHz configuration measured from the factory image on this Tab5.
	store32(i2cClock, 1<<1)
	store32(i2cBase+0x00, 0xc7)
	store32(i2cBase+0x0c, 0x31)
	store32(i2cBase+0x30, 0x31)
	store32(i2cBase+0x34, 0x63)
	store32(i2cBase+0x38, 0xc466)
	store32(i2cBase+0x40, 0xc7)
	store32(i2cBase+0x44, 0xc7)
	store32(i2cBase+0x48, 0xc7)
	store32(i2cBase+0x4c, 0xc7)
	store32(i2cBase+0x50, 0)
	store32(i2cFIFOConfig, 0x408b)
	store32(i2cBase+0x28, 0)
	store32(i2cControl, 1<<4|1<<11)
}

// configureTouchI2C switches the already initialized internal bus from the
// PI4IO expander's 100 kHz timing to the ST7121 driver's normal 400 kHz timing.
// The values below are the ESP32-P4 HAL calculation for a 40 MHz XTAL source.
func configureTouchI2C() bool {
	if !i2cWaitIdle() {
		return false
	}
	store32(i2cBase+0x00, 0x31)
	store32(i2cBase+0x30, 0x0b)
	store32(i2cBase+0x34, 0x18)
	store32(i2cBase+0x38, 0x2e1b)
	store32(i2cBase+0x40, 0x31)
	store32(i2cBase+0x44, 0x31)
	store32(i2cBase+0x48, 0x31)
	store32(i2cBase+0x4c, 0x31)
	update32(i2cControl, 0, 1<<11)
	return true
}

func i2cWrite(address byte, data []byte) bool {
	if len(data)+1 > 32 {
		return false
	}
	if !i2cWaitIdle() {
		lastI2CFailure = 8
		return false
	}
	i2cResetFIFOs()
	store32(i2cData, uint32(address)<<1)
	for index := 0; index < len(data); index++ {
		store32(i2cData, uint32(data[index]))
	}
	i2cCommand(0, i2cOperationRestart)
	i2cCommand(1, i2cOperationWrite|1<<8|uint32(len(data)+1))
	i2cCommand(2, i2cOperationStop)
	i2cCommand(3, i2cOperationEnd)
	return i2cBegin(2)
}

func i2cWriteRead(address byte, register []byte, data []byte) bool {
	if len(register)+2 > 32 || len(data) == 0 {
		return false
	}
	if !i2cWaitIdle() {
		lastI2CFailure = 8
		return false
	}

	// The P4 controller holds SCL low after END so a transaction can be
	// continued by a later command list.  The Tab5 factory driver uses that
	// facility for register reads: transmit the write address and register
	// pointer, wait for END_DETECT, then issue the repeated start and read in a
	// second TRANS_START.  ST7121 NACKs its read address when both phases are
	// submitted in a single command list.
	i2cResetFIFOs()
	store32(i2cData, uint32(address)<<1)
	for index := 0; index < len(register); index++ {
		store32(i2cData, uint32(register[index]))
	}
	i2cCommand(0, i2cOperationRestart)
	i2cCommand(1, i2cOperationWrite|1<<8|uint32(len(register)+1))
	i2cCommand(2, i2cOperationEnd)
	if !i2cBegin(2) {
		lastI2CFailure = 9
		return false
	}

	// Do not reset the FIFO or master FSM here: END deliberately preserves the
	// active bus transaction. The first phase consumed the FIFO completely, so
	// append the read address and continue with a repeated start. Drain long
	// reads while the command engine runs so the device sees one transaction.
	store32(i2cData, uint32(address)*2+1)
	if !i2cContinueRead(data) {
		lastI2CFailure = 10
		return false
	}
	return true
}

func i2cReadRegister(dataPin, clockPin int, address byte, register uint16, data []byte) bool {
	lastI2CFailure = 0
	lastStartFailure = 0
	pointer := [2]byte{byte(register >> 8), byte(register)}
	if !i2cWriteRead(address, pointer[:], data) {
		if lastI2CFailure == 0 {
			lastI2CFailure = 7
		}
		return false
	}
	return true
}

// i2cReadRegister8 performs the one-byte register-pointer transaction used by
// the Tab5's PI4IO expanders. The ST7121 uses a two-byte register pointer, so
// this deliberately remains a separate operation.
func i2cReadRegister8(dataPin, clockPin int, address, register byte) (byte, bool) {
	pointer := [1]byte{register}
	value := [1]byte{}
	ok := i2cWriteRead(address, pointer[:], value[:])
	return value[0], ok
}

func i2cWriteRegister8(dataPin, clockPin int, address, register, value byte) bool {
	data := [2]byte{register, value}
	return i2cWrite(address, data[:])
}

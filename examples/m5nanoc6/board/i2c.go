package board

const (
	ioMuxGPIO1     = uintptr(0x60090008)
	ioMuxGPIO2     = uintptr(0x6009000c)
	gpio1OutSelect = uintptr(0x60091558)
	gpio2OutSelect = uintptr(0x6009155c)

	i2cSCL = uint32(1 << 1)
	i2cSDA = uint32(1 << 2)
)

func configureOpenDrainPin(ioMux uintptr, outputSelect uintptr, pin uint32) {
	value := load32(ioMux)
	value = value &^ (gpioFunctionMask | gpioPullDown)
	store32(ioMux, value|gpioFunction|gpioInputEnable|gpioPullUp)
	store32(outputSelect, gpioOutputSignal)
	store32(gpioOutClear, pin)
	store32(gpioEnableClear, pin)
}

// ConfigureI2C configures the NanoC6 Grove connector as a software I2C bus:
// GPIO2 is SDA and GPIO1 is SCL. Outputs are open drain; the GPIO output value
// remains low and toggling output-enable either pulls a line low or releases it.
func ConfigureI2C() {
	configureOpenDrainPin(ioMuxGPIO1, gpio1OutSelect, i2cSCL)
	configureOpenDrainPin(ioMuxGPIO2, gpio2OutSelect, i2cSDA)

	// Free a slave left halfway through a byte by an interrupted transaction.
	for pulse := 0; pulse < 9; pulse++ {
		i2cPullLow(i2cSCL)
		i2cPause()
		i2cRelease(i2cSCL)
		i2cWaitHigh(i2cSCL)
		i2cPause()
	}
	i2cStop()
}

func i2cPullLow(pin uint32) {
	store32(gpioEnableSet, pin)
}

func i2cRelease(pin uint32) {
	store32(gpioEnableClear, pin)
}

func i2cHigh(pin uint32) bool {
	return load32(gpioInput)&pin != 0
}

func i2cWaitHigh(pin uint32) bool {
	for attempt := 0; attempt < 1000; attempt++ {
		if i2cHigh(pin) {
			return true
		}
		i2cPause()
	}
	return false
}

func i2cPause() {
	DelayMicroseconds(5)
}

func i2cStart() bool {
	i2cRelease(i2cSDA)
	i2cRelease(i2cSCL)
	if !i2cWaitHigh(i2cSCL) || !i2cHigh(i2cSDA) {
		return false
	}
	i2cPause()
	i2cPullLow(i2cSDA)
	i2cPause()
	i2cPullLow(i2cSCL)
	return true
}

func i2cStop() {
	i2cPullLow(i2cSDA)
	i2cPause()
	i2cRelease(i2cSCL)
	i2cWaitHigh(i2cSCL)
	i2cPause()
	i2cRelease(i2cSDA)
	i2cPause()
}

func i2cWriteByte(value byte) bool {
	mask := byte(0x80)
	for mask != 0 {
		if value&mask == 0 {
			i2cPullLow(i2cSDA)
		} else {
			i2cRelease(i2cSDA)
		}
		i2cPause()
		i2cRelease(i2cSCL)
		if !i2cWaitHigh(i2cSCL) {
			return false
		}
		i2cPause()
		i2cPullLow(i2cSCL)
		mask = mask >> 1
	}

	i2cRelease(i2cSDA)
	i2cPause()
	i2cRelease(i2cSCL)
	if !i2cWaitHigh(i2cSCL) {
		return false
	}
	i2cPause()
	acknowledged := !i2cHigh(i2cSDA)
	i2cPullLow(i2cSCL)
	return acknowledged
}

func i2cReadByte(acknowledge bool, result *uint32) bool {
	value := uint32(0)
	i2cRelease(i2cSDA)
	for bit := 0; bit < 8; bit++ {
		value = value << 1
		i2cPause()
		i2cRelease(i2cSCL)
		if !i2cWaitHigh(i2cSCL) {
			return false
		}
		if i2cHigh(i2cSDA) {
			value = value | 1
		}
		i2cPause()
		i2cPullLow(i2cSCL)
	}

	if acknowledge {
		i2cPullLow(i2cSDA)
	} else {
		i2cRelease(i2cSDA)
	}
	i2cPause()
	i2cRelease(i2cSCL)
	if !i2cWaitHigh(i2cSCL) {
		return false
	}
	i2cPause()
	i2cPullLow(i2cSCL)
	i2cRelease(i2cSDA)
	*result = value
	return true
}

func I2CWrite(address byte, data []byte) bool {
	if !i2cStart() {
		i2cStop()
		return false
	}
	if !i2cWriteByte(address << 1) {
		i2cStop()
		return false
	}
	for index := 0; index < len(data); index++ {
		if !i2cWriteByte(data[index]) {
			i2cStop()
			return false
		}
	}
	i2cStop()
	return true
}

func i2cBeginRead(address byte) bool {
	if !i2cStart() {
		i2cStop()
		return false
	}
	if !i2cWriteByte(address<<1 | 1) {
		i2cStop()
		return false
	}
	return true
}

func I2CRead(address byte, data []byte) bool {
	if !i2cBeginRead(address) {
		return false
	}
	for index := 0; index < len(data); index++ {
		value := uint32(0)
		if !i2cReadByte(index+1 < len(data), &value) {
			i2cStop()
			return false
		}
		data[index] = byte(value)
	}
	i2cStop()
	return true
}

// I2CRead6 reads the two CRC-protected three-byte words used by Sensirion
// measurement responses. Keeping the count in the function contract avoids a
// dynamic descriptor in small freestanding callers.
func I2CRead6(address byte, data *[6]uint32) bool {
	if !i2cBeginRead(address) {
		return false
	}
	for index := 0; index < 6; index++ {
		value := uint32(0)
		if !i2cReadByte(index+1 < 6, &value) {
			i2cStop()
			return false
		}
		(*data)[index] = value
	}
	i2cStop()
	return true
}

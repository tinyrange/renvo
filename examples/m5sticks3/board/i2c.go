package board

const (
	i2cSDA      = 47
	i2cSCL      = 48
	pmicAddress = byte(0x6e)
)

func i2cDelay() {
	DelayMicroseconds(2)
}

func i2cRelease(pin int) {
	enableGPIO(pin, false)
	i2cDelay()
}

func i2cLow(pin int) {
	setGPIO(pin, false)
	enableGPIO(pin, true)
	i2cDelay()
}

func i2cStart() {
	i2cRelease(i2cSDA)
	i2cRelease(i2cSCL)
	i2cLow(i2cSDA)
	i2cLow(i2cSCL)
}

func i2cStop() {
	i2cLow(i2cSDA)
	i2cRelease(i2cSCL)
	i2cRelease(i2cSDA)
}

func i2cWriteByte(value byte) bool {
	for bit := 7; bit >= 0; bit-- {
		i2cLow(i2cSCL)
		if value&(byte(1)<<uint(bit)) != 0 {
			i2cRelease(i2cSDA)
		} else {
			i2cLow(i2cSDA)
		}
		i2cRelease(i2cSCL)
	}
	i2cLow(i2cSCL)
	i2cRelease(i2cSDA)
	i2cRelease(i2cSCL)
	ack := load32(gpioInput1)&pinBit(i2cSDA) == 0
	i2cLow(i2cSCL)
	return ack
}

func i2cReadByte() byte {
	var value byte
	i2cRelease(i2cSDA)
	for bit := 7; bit >= 0; bit-- {
		i2cLow(i2cSCL)
		i2cRelease(i2cSCL)
		if load32(gpioInput1)&pinBit(i2cSDA) != 0 {
			value |= byte(1) << uint(bit)
		}
	}
	// NACK the final byte.
	i2cLow(i2cSCL)
	i2cRelease(i2cSDA)
	i2cRelease(i2cSCL)
	i2cLow(i2cSCL)
	return value
}

func pmicWrite(register byte, value byte) bool {
	i2cStart()
	ok := i2cWriteByte(pmicAddress << 1)
	ok = i2cWriteByte(register) && ok
	ok = i2cWriteByte(value) && ok
	i2cStop()
	return ok
}

func pmicRead(register byte) (byte, bool) {
	i2cStart()
	ok := i2cWriteByte(pmicAddress << 1)
	ok = i2cWriteByte(register) && ok
	i2cStart()
	ok = i2cWriteByte(pmicAddress<<1|1) && ok
	value := i2cReadByte()
	i2cStop()
	return value, ok
}

func pmicUpdate(register byte, mask byte, set bool) bool {
	value, ok := pmicRead(register)
	if !ok {
		return false
	}
	if set {
		value |= mask
	} else {
		value &^= mask
	}
	return pmicWrite(register, value)
}

// EnableLCDPower enables only the M5PM1 G2/L3B output used by the LCD. It
// verifies the PMIC identity before making the same masked register changes as
// M5Stack's StickS3 initialization; it does not alter any voltage setting.
func EnableLCDPower() bool {
	configureGPIO(i2cSDA, true)
	configureGPIO(i2cSCL, true)
	i2cRelease(i2cSDA)
	i2cRelease(i2cSCL)
	idLow, ok := pmicRead(0x00)
	if !ok {
		return false
	}
	idHigh, ok := pmicRead(0x01)
	if !ok || idLow != 0x50 || idHigh != 0x20 {
		return false
	}
	if !pmicUpdate(0x16, 1<<2, false) {
		return false
	}
	if !pmicUpdate(0x10, 1<<2, true) {
		return false
	}
	if !pmicUpdate(0x13, 1<<2, false) {
		return false
	}
	if !pmicUpdate(0x11, 1<<2, true) {
		return false
	}
	if !pmicWrite(0x09, 0) {
		return false
	}
	DelayMilliseconds(10)
	return true
}

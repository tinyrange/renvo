package board

import (
	"renvo.dev/device/esp32s3"
	"renvo.dev/device/i2c"
)

const pmicAddress = uint16(0x6e)

var pmicController = i2c.NewBitBang(esp32s3.GPIO(47), esp32s3.GPIO(48), &Clock, 250000)
var pmicPort = i2c.DefinePort(&pmicController, &Clock)

func pmicWrite(bus *i2c.Bus, register byte, value byte) bool {
	return bus.Write(pmicAddress, []byte{register, value}) == nil
}

func pmicRead(bus *i2c.Bus, register byte) (byte, bool) {
	data := []byte{0}
	if err := bus.Tx(pmicAddress, []byte{register}, data); err != nil {
		return 0, false
	}
	return data[0], true
}

func pmicUpdate(bus *i2c.Bus, register byte, mask byte, set bool) bool {
	value, ok := pmicRead(bus, register)
	if !ok {
		return false
	}
	if set {
		value |= mask
	} else {
		value &^= mask
	}
	return pmicWrite(bus, register, value)
}

// enableLCDPower enables only the M5PM1 G2/L3B output used by the LCD. It
// verifies the PMIC identity before making the same masked register changes as
// M5Stack's StickS3 initialization; it does not alter any voltage setting.
func enableLCDPower() bool {
	bus := i2c.New(pmicPort)
	idLow, ok := pmicRead(bus, 0x00)
	if !ok {
		return false
	}
	idHigh, ok := pmicRead(bus, 0x01)
	if !ok || idLow != 0x50 || idHigh != 0x20 {
		return false
	}
	if !pmicUpdate(bus, 0x16, 1<<2, false) {
		return false
	}
	if !pmicUpdate(bus, 0x10, 1<<2, true) {
		return false
	}
	if !pmicUpdate(bus, 0x13, 1<<2, false) {
		return false
	}
	if !pmicUpdate(bus, 0x11, 1<<2, true) {
		return false
	}
	if !pmicWrite(bus, 0x09, 0) {
		return false
	}
	Clock.DelayMilliseconds(10)
	return true
}

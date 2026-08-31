//go:build m5sticks3

package board

import (
	"renvo.dev/device/esp32s3"
	"renvo.dev/device/i2c"
	"renvo.dev/device/power/m5pm1"
)

var pmicController = i2c.NewBitBang(esp32s3.GPIO(47), esp32s3.GPIO(48), &Clock, 250000)
var pmicPort = i2c.DefinePort(&pmicController, &Clock)
var pmic = m5pm1.New(i2c.New(pmicPort))

// enableLCDPower enables only the M5PM1 G2/L3B output used by the LCD. It
// verifies the PMIC identity and establishes the output latch before enabling
// the driver; it does not alter any voltage setting.
func enableLCDPower() bool {
	if err := pmic.Initialize(); err != nil {
		return false
	}
	if err := pmic.ConfigureOutput(m5pm1.Pin2, true); err != nil {
		return false
	}
	Clock.DelayMilliseconds(10)
	return true
}

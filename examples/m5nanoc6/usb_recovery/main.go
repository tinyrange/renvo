// usb_recovery proves that a wedged raw-PHY attempt watchdog-resets the C6 and
// enters a retained recovery window with USB Serial/JTAG restored.
package main

import (
	board "renvo.dev/device/board/m5nanoc6"
	"renvo.dev/device/esp32c6"
)

func main() {
	if esp32c6.USBRecoveryPending() {
		board.BlueLED.Set(true)
	}
	esp32c6.OpenUSBRecoveryWindow()
	esp32c6.ArmUSBRecovery()
	phy := esp32c6.NewUSBPHY(0)
	phy.Takeover()
	for {
	}
}

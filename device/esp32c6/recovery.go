package esp32c6

import "renvo.dev/device/mmio"

const (
	usbRecoveryStore = uintptr(0x600b1024) // LP_AON_STORE9_REG
	usbRecoveryMagic = uint32(0x52454e56)  // "RENV"

	timerGroup0WDTConfig0 = uintptr(0x60008048)
	timerGroup0WDTConfig1 = uintptr(0x6000804c)
	timerGroup0WDTStage0  = uintptr(0x60008050)
	timerGroup0WDTFeed    = uintptr(0x60008060)
	timerGroup0WDTProtect = uintptr(0x60008064)
	timerGroup0WDTKey     = uint32(0x50d83aa1)
)

// USBRecoveryPending reports whether the preceding raw-PHY attempt ended in a
// watchdog reset. The marker lives in an always-on store register retained by
// an HP-system reset.
func USBRecoveryPending() bool {
	return mmio.Load32(usbRecoveryStore) == usbRecoveryMagic
}

// OpenUSBRecoveryWindow leaves the reset-default USB Serial/JTAG peripheral in
// control for five seconds after a failed raw-PHY attempt. A waiting flashing
// tool can use its modem-control lines to enter the ROM loader during this
// window. Successful boots do not wait.
func OpenUSBRecoveryWindow() {
	if USBRecoveryPending() {
		timer := SystemTimer{}
		timer.DelayMilliseconds(5000)
	}
}

// ArmUSBRecovery marks a raw-PHY attempt in progress and arms Timer Group 0's
// main watchdog for an HP-system reset after twenty seconds. With an 80 MHz
// source, the 20000 divider gives one watchdog tick per 500 microseconds.
func ArmUSBRecovery() {
	mmio.Store32(usbRecoveryStore, usbRecoveryMagic)
	mmio.Store32(timerGroup0WDTProtect, timerGroup0WDTKey)
	mmio.Store32(timerGroup0WDTConfig0, 0)
	mmio.Store32(timerGroup0WDTConfig1, 20000<<16|1)
	mmio.Store32(timerGroup0WDTStage0, 40000)
	mmio.Store32(timerGroup0WDTFeed, 1)
	mmio.Store32(timerGroup0WDTConfig0, 1<<31|3<<29|1<<22)
	mmio.Store32(timerGroup0WDTProtect, 0)
}

// CompleteUSBRecovery clears the retained attempt marker and disarms the
// watchdog after the raw PHY has been returned to USB Serial/JTAG.
func CompleteUSBRecovery() {
	mmio.Store32(timerGroup0WDTProtect, timerGroup0WDTKey)
	mmio.Store32(timerGroup0WDTConfig0, 0)
	mmio.Store32(timerGroup0WDTProtect, 0)
	mmio.Store32(usbRecoveryStore, 0)
}

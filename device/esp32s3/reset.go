package esp32s3

import "renvo.dev/device/mmio"

const (
	rtcOption1       = uintptr(0x6000812c)
	rtcWDTConfig0    = uintptr(0x60008098)
	rtcWDTConfig1    = uintptr(0x6000809c)
	rtcWDTWriteGuard = uintptr(0x600080b0)
	rtcWDTKey        = uint32(0x50d83aa1)
	rtcTraceStore    = uintptr(0x50001f00)
)

var usbRecoveryTrace [4]uint32

// ArmUSBRecovery records an initial trace stage, requests download mode on the
// next reset, and arms the RTC watchdog before controller initialization. The
// bounded timeout is intentionally short while qualifying a controller build.
func ArmUSBRecovery(stage uint32) {
	mmio.Register32(rtcOption1).Replace(0, 1)
	mmio.Store32(rtcTraceStore, stage)
	mmio.Store32(rtcWDTWriteGuard, rtcWDTKey)
	mmio.Store32(rtcWDTConfig1, 20000)
	mmio.Store32(rtcWDTConfig0, 1<<31|5<<28|1<<8|2)
	mmio.Store32(rtcWDTWriteGuard, 0)
}

// USBRecoveryStage stores the most recent native-USB bring-up checkpoint in
// RTC retention memory for inspection after a watchdog recovery.
func USBRecoveryStage(stage uint32) {
	usbRecoveryTrace[0] = stage
	mmio.Store32(rtcTraceStore, stage)
}

// USBRecoveryData stores one of three auxiliary trace words.
func USBRecoveryData(index uint8, value uint32) {
	if index >= 1 && index <= 3 {
		usbRecoveryTrace[index] = value
		mmio.Store32(rtcTraceStore+uintptr(index)*4, value)
	}
}

func USBRecoveryTrace(index uint8) uint32 {
	if index < 4 {
		return usbRecoveryTrace[index]
	}
	return 0
}

// CompleteUSBRecovery disarms the watchdog once the host has configured USB.
func CompleteUSBRecovery() {
	mmio.Register32(rtcOption1).Replace(1, 0)
	mmio.Store32(rtcWDTWriteGuard, rtcWDTKey)
	mmio.Store32(rtcWDTConfig0, 0)
	mmio.Store32(rtcWDTWriteGuard, 0)
}

// ResetToUSBDownload asks the ESP32-S3 ROM to enter its fixed USB
// Serial/JTAG downloader, then resets through the RTC watchdog. It is the
// recovery path for applications that have routed the shared PHY to DWC2.
func ResetToUSBDownload() {
	mmio.Register32(rtcOption1).Replace(0, 1)
	mmio.Store32(rtcWDTWriteGuard, rtcWDTKey)
	mmio.Store32(rtcWDTConfig1, 2000)
	mmio.Store32(rtcWDTConfig0, 1<<31|5<<28|1<<8|2)
	mmio.Store32(rtcWDTWriteGuard, 0)
	for {
	}
}

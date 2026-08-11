// Package board describes the hardware attached to an M5Stack StickS3.
package board

import (
	"renvo.dev/device/esp32s3"
	"renvo.dev/device/gpio"
	"renvo.dev/device/usb"
	"renvo.dev/std/graphics"
)

// FrontButton is the active-low front button on GPIO11.
var FrontButton = gpio.NewButton(esp32s3.GPIO(11), gpio.PullNone, true)

// SideButton is the active-low side button on GPIO12.
var SideButton = gpio.NewButton(esp32s3.GPIO(12), gpio.PullNone, true)

// Clock is the board monotonic clock and busy-wait timer.
var Clock = esp32s3.SystemTimer{}

var usbController = esp32s3.DWC2{}

// USB is the native USB-C device connection on GPIO19/GPIO20.
var USB = usb.DefinePort(&usbController)

// USBDownload provides a recoverable way out of an invalid native-USB image.
// Calling Enter resets into the fixed ESP ROM USB Serial/JTAG downloader.
var USBDownload = DownloadMode{}

type DownloadMode struct{}

func (*DownloadMode) Arm()                     { esp32s3.ArmUSBRecovery(1) }
func (*DownloadMode) Complete()                { esp32s3.CompleteUSBRecovery() }
func (*DownloadMode) Enter()                   { esp32s3.ResetToUSBDownload() }
func (*DownloadMode) Return()                  { usbController.ReturnToSerialJTAG() }
func (*DownloadMode) Trace(index uint8) uint32 { return esp32s3.USBRecoveryTrace(index) }

// Screen exposes the board-attached ST7789 without leaking its SPI and GDMA
// wiring.
type Screen struct{}

// Display is the attached 135x240 LCD.
var Display = Screen{}

func (*Screen) Width() int  { return DisplayWidth }
func (*Screen) Height() int { return DisplayHeight }
func (*Screen) Initialize() bool {
	return InitializeDisplay()
}
func (*Screen) PresentSurface2x(surface *graphics.Surface) bool {
	return PresentSurface2x(surface)
}
func (*Screen) DrawLineDiagnostic() bool { return DrawLineDiagnostic() }
func (*Screen) DrawButtonRectangle(index int, visible bool) {
	DrawButtonRectangle(index, visible)
}

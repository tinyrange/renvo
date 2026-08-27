package bios_test

import "renvo.dev/device/bios"

// WriteConsole uses the firmware teletype service and needs no runtime console.
func ExampleWriteConsole() {
	bios.WriteConsole("Hello from Renvo!\r\n")
}

func ExampleReadKey() {
	ascii, scan := bios.ReadKey()
	if ascii == 0 {
		bios.WriteConsole("extended key scan code received\r\n")
	}
	_, _ = ascii, scan
}

func ExampleReadSectors() {
	const (
		loadSegment = 0x0000
		loadOffset  = 0x7e00
	)
	drive := bios.BootDrive()
	if err := bios.ReadSectors(drive, 1, 1, loadSegment, loadOffset); err != nil {
		bios.WriteConsole("disk read failed\r\n")
	}
}

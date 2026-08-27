package main

import (
	"fmt"

	"renvo.dev/device/uefi"
)

func main() {
	uefi.ClearScreen()
	fmt.Println("Renvo UEFI application")
	fmt.Println()
	vendor := uefi.FirmwareVendor()
	if vendor == "" {
		vendor = "unknown firmware"
	}
	fmt.Println("Firmware: " + vendor)
	fmt.Println("The system table and text output protocol are working.")
	fmt.Println()
	fmt.Println("Press any key to return to the firmware.")
	for {
		if _, status := uefi.ReadKey(); status == uefi.Success {
			return
		}
		uefi.Stall(10000)
	}
}

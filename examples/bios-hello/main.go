package main

import "renvo.dev/device/bios"

func main() {
	print("Renvo PC BIOS application\n")
	drive := bios.BootDrive()
	if bios.ExtensionsAvailable(drive) {
		print("INT 13h extensions available\n")
	} else {
		print("Using legacy CHS disk services\n")
	}
	start, _ := bios.Ticks()
	for {
		now, _ := bios.Ticks()
		if now != start {
			break
		}
	}
	print("Firmware timer is running; press a key to halt.\n")
	bios.ReadKey()
}

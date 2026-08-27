package main

import (
	"fmt"

	"renvo.dev/device/uefi"
)

func fail(operation string, status uefi.Status) {
	fmt.Println(operation + ": " + status.Error())
}

func main() {
	fmt.Println("UEFI Simple File System demo")
	root, status := uefi.OpenVolume()
	if status.Failed() {
		fail("OpenVolume", status)
		return
	}
	defer root.Close()

	file, status := root.Open("RENVO.TXT",
		uefi.FileModeRead|uefi.FileModeWrite|uefi.FileModeCreate, uefi.FileArchive)
	if status.Failed() {
		fail("Open RENVO.TXT", status)
		return
	}
	message := []byte("Written by a Renvo UEFI application.\r\n")
	written, status := file.Write(message)
	if status.Failed() {
		fail("Write RENVO.TXT", status)
		file.Close()
		return
	}
	file.Flush()
	file.Close()

	fmt.Println("Wrote RENVO.TXT (" + decimal(written) + " bytes).")
}

func decimal(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	at := len(digits)
	for value > 0 {
		at--
		digits[at] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[at:])
}

package main

import "renvo.dev/device/uefi"

func qemuExit(code uint32)

func main() {
	table := uefi.CurrentSystemTable()
	if !table.Valid() {
		qemuExit(11)
		return
	}
	console := table.ConsoleOutput()
	if console == nil {
		qemuExit(12)
		return
	}
	if console.OutputString == 0 {
		qemuExit(13)
		return
	}
	print("PASS\n")
	status := uefi.Success
	if uefi.Stall(1000).Failed() {
		qemuExit(2)
		return
	}
	if _, status = uefi.GraphicsOutput(); status.Failed() {
		qemuExit(3)
		return
	}
	root, status := uefi.OpenVolume()
	if status.Failed() {
		qemuExit(4)
		return
	}
	file, status := root.Open("EFI\\BOOT\\BOOTX64.EFI", uefi.FileModeRead, 0)
	if status.Failed() {
		qemuExit(5)
		return
	}
	header := make([]byte, 2)
	if _, status = file.Read(header); status.Failed() || string(header) != "MZ" {
		qemuExit(6)
		return
	}
	file.Close()
	root.Close()
	qemuExit(42)
}

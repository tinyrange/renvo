package main

import (
	"renvo.dev/device/board"
	"renvo.dev/device/esp32p4"
	"unsafe"
)

// This test never opens the SD card. It exercises the actual internal-SRAM
// allocation and clock transition, then runs the shared C core regressions.
func host_memory(size int32) *byte {
	address, err := board.ReserveInternalMemory(uintptr(size), 64)
	if err != nil {
		print("FAIL reserve: ", err.Error(), "\n")
		return nil
	}
	return (*byte)(unsafe.Pointer(address))
}

func main() {
	if err := esp32p4.UseFullPLLClock(); err != nil {
		print("FAIL clock: ", err.Error(), "\n")
		return
	}
	var timer esp32p4.SystemTimer
	timer.DelayMilliseconds(1000) // Allow a serial monitor to attach after flashing.
	if _, err := board.ReserveInternalMemory(1, 3); err == nil {
		print("FAIL invalid alignment\n")
		return
	}
	print("PDP CORE P4 INTERNAL SRAM START\n")
	if run_test() != 0 {
		return
	}
	if _, err := board.ReserveInternalMemory(4096, 64); err != nil {
		print("FAIL remaining SRAM: ", err.Error(), "\n")
		return
	}
	if _, err := board.ReserveInternalMemory(1, 1); err == nil {
		print("FAIL SRAM overflow\n")
		return
	}
	print("PASS SRAM bounds and alignment\n")
}

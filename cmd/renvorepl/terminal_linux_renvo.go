//go:build renvo && linux

package main

import "unsafe"

const (
	replLinuxTCGETS = 0x5401
	replLinuxTCSETS = 0x5402

	replLinuxBRKINT = 0x0002
	replLinuxINPCK  = 0x0010
	replLinuxISTRIP = 0x0020
	replLinuxICRNL  = 0x0100
	replLinuxIXON   = 0x0400

	replLinuxCSIZE  = 0x0030
	replLinuxCS8    = 0x0030
	replLinuxPARENB = 0x0100

	replLinuxISIG   = 0x0001
	replLinuxICANON = 0x0002
	replLinuxECHO   = 0x0008
	replLinuxIEXTEN = 0x8000

	replLinuxTermiosSize = 64
	replLinuxCCOffset    = 17
	replLinuxVTIME       = 5
	replLinuxVMIN        = 6
)

var replLinuxSavedTermios []byte

func replTerminalEnable() bool {
	saved := make([]byte, replLinuxTermiosSize)
	if replLinuxIoctl(0, replLinuxTCGETS, saved) < 0 {
		return false
	}
	raw := make([]byte, len(saved))
	copy(raw, saved)

	inputFlags := replLinuxUint32(raw, 0)
	inputFlags = replLinuxClearBits(inputFlags, replLinuxBRKINT|replLinuxICRNL|replLinuxINPCK|replLinuxISTRIP|replLinuxIXON)
	replLinuxPutUint32(raw, 0, inputFlags)

	controlFlags := replLinuxUint32(raw, 8)
	controlFlags = replLinuxClearBits(controlFlags, replLinuxCSIZE|replLinuxPARENB)
	controlFlags = controlFlags | replLinuxCS8
	replLinuxPutUint32(raw, 8, controlFlags)

	localFlags := replLinuxUint32(raw, 12)
	localFlags = replLinuxClearBits(localFlags, replLinuxECHO|replLinuxICANON|replLinuxIEXTEN|replLinuxISIG)
	replLinuxPutUint32(raw, 12, localFlags)

	raw[replLinuxCCOffset+replLinuxVTIME] = 0
	raw[replLinuxCCOffset+replLinuxVMIN] = 1
	if replLinuxIoctl(0, replLinuxTCSETS, raw) < 0 {
		return false
	}
	replLinuxSavedTermios = saved
	return true
}

func replTerminalDisable() {
	if len(replLinuxSavedTermios) == 0 {
		return
	}
	replLinuxIoctl(0, replLinuxTCSETS, replLinuxSavedTermios)
	replLinuxSavedTermios = nil
}

func replLinuxIoctl(fd int, request int, data []byte) int {
	if len(data) == 0 {
		return -1
	}
	return renvo_runtime_Syscall(replLinuxIoctlNumber(), fd, request, int(unsafe.Pointer(&data[0])), 0, 0, 0)
}

func replLinuxUint32(data []byte, offset int) int {
	value := int(data[offset])
	value = value | int(data[offset+1])<<8
	value = value | int(data[offset+2])<<16
	value = value | int(data[offset+3])<<24
	return value
}

func replLinuxPutUint32(data []byte, offset int, value int) {
	data[offset] = byte(value)
	data[offset+1] = byte(value >> 8)
	data[offset+2] = byte(value >> 16)
	data[offset+3] = byte(value >> 24)
}

func replLinuxClearBits(value int, bits int) int {
	return value ^ (value & bits)
}

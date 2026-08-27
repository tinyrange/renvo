package main

import (
	"renvo.dev/device/mmio"
	"renvo.dev/device/rp2"
)

const (
	commandInfo      = byte(1)
	commandBegin     = byte(2)
	commandWrite     = byte(3)
	commandCommit    = byte(4)
	commandWriteFast = byte(5)

	protocolMajor  = byte(1)
	protocolMinor  = byte(0)
	monitorVersion = uint32(1 << 16) // 1.0.0

	capabilityFastWrite = uint32(1)
	chipRP2040          = uint32(0x2040)
	chipRP2350          = uint32(0x2350)

	reloadStart = uintptr(0x20010000)
	reloadEnd   = uintptr(0x20040000)
	stackTop    = uint32(0x20040000)

	sioBase   = uintptr(0xd0000000)
	fifoState = sioBase + 0x50
	fifoWrite = sioBase + 0x54
	fifoRead  = sioBase + 0x58
)

var generation uint32
var clientVersion uint32
var handshakeCompatible bool

func load32(data []byte, offset int) uint32 {
	return uint32(data[offset]) | uint32(data[offset+1])<<8 |
		uint32(data[offset+2])<<16 | uint32(data[offset+3])<<24
}

func store32(data []byte, offset int, value uint32) {
	data[offset] = byte(value)
	data[offset+1] = byte(value >> 8)
	data[offset+2] = byte(value >> 16)
	data[offset+3] = byte(value >> 24)
}

func isRP2350() bool { return (mmio.Load32(0xe000ed00)>>4)&0xfff == 0xd21 }

func chipIdentifier() uint32 {
	if isRP2350() {
		return chipRP2350
	}
	return chipRP2040
}

func psm() (uintptr, uint32) {
	if isRP2350() {
		return 0x40018000, 1 << 24
	}
	return 0x40010000, 1 << 16
}

func fifoPop() uint32 {
	for mmio.Load32(fifoState)&1 == 0 {
	}
	return mmio.Load32(fifoRead)
}

func fifoPush(value uint32) {
	for mmio.Load32(fifoState)&2 == 0 {
	}
	mmio.Store32(fifoWrite, value)
}

func fifoDrain() {
	for mmio.Load32(fifoState)&1 != 0 {
		_ = mmio.Load32(fifoRead)
	}
}

func resetApplicationCore() {
	base, mask := psm()
	forceOff := base + 4
	mmio.Store32(forceOff+0x2000, mask)
	for mmio.Load32(forceOff)&mask == 0 {
	}
	fifoDrain()
}

func coreLaunchSequence(entry uint32) [6]uint32 {
	return [6]uint32{0, 0, 1, uint32(reloadStart), stackTop, entry | 1}
}

func launchApplication(entry uint32) {
	base, mask := psm()
	mmio.Store32(base+4+0x3000, mask)
	// Match the ROM handshake used by pico_multicore: two synchronizing zeros,
	// the launch command, vector table, stack pointer, and Thumb entry point.
	// Core 1 echoes every word. A mismatch restarts the sequence.
	commands := coreLaunchSequence(entry)
	index := 0
	for index < len(commands) {
		command := commands[index]
		if command == 0 {
			fifoDrain()
			// Freestanding print emits SEV, waking a ROM core parked in WFE.
			print(" ")
		}
		fifoPush(command)
		if fifoPop() == command {
			index++
		} else {
			index = 0
		}
	}
}

func reply(usb *rp2.USBDevice, operation byte, status byte) {
	var packet [64]byte
	packet[0], packet[1], packet[2], packet[3] = 'R', 'N', 'V', '2'
	packet[4], packet[5] = operation, status
	packet[6], packet[7] = protocolMajor, protocolMinor
	store32(packet[:], 8, generation)
	store32(packet[:], 12, uint32(reloadStart))
	store32(packet[:], 16, uint32(reloadEnd))
	store32(packet[:], 20, capabilityFastWrite)
	store32(packet[:], 24, monitorVersion)
	store32(packet[:], 28, chipIdentifier())
	store32(packet[:], 32, clientVersion)
	for !usb.WritePacket(packet[:]) {
		usb.Poll()
	}
}

func handle(usb *rp2.USBDevice, packet []byte, count int) {
	if count < 12 || packet[0] != 'R' || packet[1] != 'N' ||
		packet[2] != 'V' || packet[3] != '2' {
		return
	}
	operation := packet[4]
	if operation == commandInfo {
		clientVersion = load32(packet, 8)
		handshakeCompatible = packet[6] == protocolMajor
		reply(usb, operation, 0)
		return
	}
	if !handshakeCompatible {
		reply(usb, operation, 2)
		return
	}
	if operation == commandBegin {
		resetApplicationCore()
		reply(usb, operation, 0)
		return
	}
	if operation == commandWrite || operation == commandWriteFast {
		address := uintptr(load32(packet, 8))
		length := count - 12
		if address&3 != 0 || length&3 != 0 || address < reloadStart || address > reloadEnd ||
			uintptr(length) > reloadEnd-address {
			if operation == commandWrite {
				reply(usb, operation, 1)
			}
			return
		}
		for index := 0; index < length; index += 4 {
			mmio.Store32(address+uintptr(index), load32(packet, 12+index))
		}
		if operation == commandWrite {
			reply(usb, operation, 0)
		}
		return
	}
	if operation == commandCommit {
		entry := load32(packet, 8)
		if uintptr(entry) < reloadStart || uintptr(entry) >= reloadEnd {
			reply(usb, operation, 1)
			return
		}
		generation = generation + 1
		reply(usb, operation, 0)
		launchApplication(entry)
		return
	}
	reply(usb, operation, 1)
}

func main() {
	var usb rp2.USBDevice
	var packet [64]byte
	rp2.ConfigureUSBClock()
	usb.Start()
	for {
		usb.Poll()
		if count := usb.ReadPacket(packet[:]); count != 0 {
			handle(&usb, packet[:], count)
		}
	}
}

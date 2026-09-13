// A static-address TCP echo endpoint for a direct Mac-to-Unit-PoE-P4 cable.
package main

import (
	"renvo.dev/device/esp32p4"
	"renvo.dev/device/tcpip"
	"renvo.dev/internal/arena"
)

func main() {
	var timer esp32p4.SystemTimer
	var eth esp32p4.Ethernet
	server := tcpip.Echo{MAC: [6]byte{2, 0x52, 0x4e, 0, 0, 4}, IP: [4]byte{169, 254, 180, 4}, Port: 4242}
	err := eth.Configure(esp32p4.EthernetConfig{MAC: server.MAC, MDC: 31, MDIO: 52, Reset: 51, PHY: 1, DMA: 0x4ff41000})
	if err != nil {
		for {
			print(err.Error(), "\n")
			timer.DelayMilliseconds(1000)
		}
	}
	print("RENVO POE-P4 TCP READY 169.254.180.4:4242\n")
	var frame [1514]byte
	// Accumulate elapsed ticks so the millisecond clock survives the 32-bit
	// hardware timer wrapping every 268 seconds.
	last := timer.Ticks()
	var millis, remainder uint32
	scratch := arena.Mark()
	for {
		ticks := timer.Ticks()
		delta := ticks - last
		last = ticks
		remainder += delta
		millis += remainder / 16000
		remainder %= 16000
		n := eth.Receive(frame[:])
		if n > 0 {
			if reply := server.Handle(frame[:n], millis); len(reply) > 0 {
				if err := eth.Send(reply); err != nil {
					print(err.Error(), "\n")
				}
			}
		}
		if reply := server.Poll(millis); len(reply) > 0 {
			if err := eth.Send(reply); err != nil {
				print(err.Error(), "\n")
			}
		}
		// Renvo uses an arena: reclaim temporary slice headers/lowering storage.
		// The endpoint copies pending data into its own fixed buffer, and Send
		// completes DMA before returning, so no per-poll storage escapes.
		arena.Rewind(scratch)
	}
}

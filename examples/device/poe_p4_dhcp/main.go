// A DHCP-configured version of the Unit PoE-P4 TCP echo endpoint.
package main

import (
	"fmt"
	"renvo.dev/device/esp32p4"
	"renvo.dev/device/tcpip"
	"renvo.dev/internal/arena"
)

func main() {
	var timer esp32p4.SystemTimer
	var eth esp32p4.Ethernet
	client := tcpip.DHCP{MAC: esp32p4.BaseMAC(), Hostname: "renvo-poe-p4"}
	server := tcpip.Echo{MAC: client.MAC, Port: 4242}
	var frame [1514]byte
	scratch := arena.Mark()
	for {
		err := eth.Configure(esp32p4.EthernetConfig{MAC: client.MAC, MDC: 31, MDIO: 52, Reset: 51, PHY: 1, DMA: 0x4ff41000})
		arena.Rewind(scratch)
		if err == nil {
			break
		}
		print(err.Error(), "\n")
		timer.DelayMilliseconds(2000)
	}
	fmt.Printf("RENVO DHCP MAC %x:%x:%x:%x:%x:%x hostname renvo-poe-p4\n", client.MAC[0], client.MAC[1], client.MAC[2], client.MAC[3], client.MAC[4], client.MAC[5])
	client.Start(0, timer.Ticks())
	last := timer.Ticks()
	var millis, remainder, lastLink, lastReport uint32
	link, configured := true, false
	var applied [4]byte
	for {
		ticks := timer.Ticks()
		delta := ticks - last
		last = ticks
		remainder += delta
		millis += remainder / 16000
		remainder %= 16000
		if millis-lastLink >= 500 {
			lastLink = millis
			status, err := eth.ReadPHY(1)
			up := err == nil && status&4 != 0
			if up != link {
				link = up
				if link {
					client.Start(millis, ticks)
					print("LINK UP: DHCP acquiring\n")
				} else {
					client.Stop()
					print("LINK DOWN: address removed\n")
				}
			}
		}
		n := eth.Receive(frame[:])
		if n > 0 && link {
			if reply := client.Handle(frame[:n], millis); len(reply) > 0 {
				send(&eth, reply)
			}
			if client.Ready() && configured && client.Lease.IP == applied {
				if reply := server.Handle(frame[:n], millis); len(reply) > 0 {
					send(&eth, reply)
				}
			}
		}
		if reply := client.Poll(millis); len(reply) > 0 {
			send(&eth, reply)
		}
		if !client.Ready() {
			if configured {
				configured = false
				server = tcpip.Echo{MAC: client.MAC, Port: 4242}
				print("DHCP address removed; acquiring\n")
			}
		} else {
			if !configured || applied != client.Lease.IP {
				applied = client.Lease.IP
				configured = true
				server = tcpip.Echo{MAC: client.MAC, IP: applied, Port: 4242}
				lastReport = millis - 10000
			}
			if reply := server.Poll(millis); len(reply) > 0 {
				send(&eth, reply)
			}
		}
		if millis-lastReport >= 10000 {
			lastReport = millis
			if client.Ready() {
				ip := client.Lease.IP
				mask := client.Lease.Mask
				gw := client.Lease.Router
				fmt.Printf("DHCP TCP READY %d.%d.%d.%d:4242 mask %d.%d.%d.%d gateway %d.%d.%d.%d lease %d seconds ACKs %d\n", ip[0], ip[1], ip[2], ip[3], mask[0], mask[1], mask[2], mask[3], gw[0], gw[1], gw[2], gw[3], client.Lease.Seconds, client.ACKs)
			} else {
				print("DHCP waiting for lease\n")
			}
		}
		arena.Rewind(scratch)
	}
}

func send(eth *esp32p4.Ethernet, frame []byte) {
	if err := eth.Send(frame); err != nil {
		print(err.Error(), "\n")
	}
}

// A DHCP web server for the M5Stack Unit PoE-P4.
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
	var server tcpip.HTTP
	var frame [1514]byte
	server.Configure(client.MAC, [4]byte{}, 80)
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
	client.Start(0, timer.Ticks())
	fmt.Printf("HTTP MAC %x:%x:%x:%x:%x:%x arena %d\n", client.MAC[0], client.MAC[1], client.MAC[2], client.MAC[3], client.MAC[4], client.MAC[5], scratch)
	last := timer.Ticks()
	var millis, remainder, lastLink, lastReport, uptime, uptimeMS uint32
	link, configured := true, false
	var applied [4]byte
	for {
		ticks := timer.Ticks()
		delta := ticks - last
		last = ticks
		remainder += delta
		elapsed := remainder / 16000
		remainder %= 16000
		millis += elapsed
		uptimeMS += elapsed
		uptime += uptimeMS / 1000
		uptimeMS %= 1000
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
					print("LINK DOWN\n")
				}
			}
		}
		server.Status = tcpip.HTTPStatus{MAC: client.MAC, IP: client.Lease.IP, Mask: client.Lease.Mask, Gateway: client.Lease.Router, UptimeSeconds: uptime, LeaseSeconds: client.Lease.Seconds, LeaseACKs: client.ACKs}
		n := eth.Receive(frame[:])
		if n > 0 && link {
			if reply := client.Handle(frame[:n], millis); len(reply) > 0 {
				send(&eth, reply)
			}
			if client.Ready() && configured && applied == client.Lease.IP {
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
				server.Configure(client.MAC, [4]byte{}, 80)
				print("DHCP address removed\n")
			}
		} else {
			if !configured || applied != client.Lease.IP {
				applied = client.Lease.IP
				configured = true
				server.Configure(client.MAC, applied, 80)
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
				fmt.Printf("HTTP READY http://%d.%d.%d.%d/ uptime %d seconds requests %d\n", ip[0], ip[1], ip[2], ip[3], uptime, server.Requests)
			} else {
				fmt.Printf("DHCP waiting for lease RX %d dropped %d ACKs %d conflicts %d\n", eth.Received, eth.Dropped, client.ACKs, client.Conflicts)
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

# Unit PoE-P4 TCP bring-up

A Renvo-built, freestanding TCP echo endpoint for the M5Stack Unit PoE-P4.
The connected ESP32-P4 v1.3 runs the existing `esp32p4/riscv32` target and
retains its factory second-stage bootloader. No ESP-IDF application or network
stack is linked into the application.

Connect the download USB-C port to the Mac for power/flashing, and connect the
RJ45 port directly to a USB Ethernet adapter. This example uses:

| Setting | Value |
| --- | --- |
| Board IPv4 | `169.254.180.4/16` (static; no address conflict detection) |
| TCP port | `4242` |
| Example MAC | `02:52:4e:00:00:04` (locally administered) |
| PHY | IP101, management address 1 |
| MDC / MDIO / reset | GPIO31 / GPIO52 / GPIO51 |
| RXDV / RXD0 / RXD1 | GPIO28 / GPIO29 / GPIO30 |
| TXD0 / TXD1 / TXEN / reference clock input | GPIO34 / GPIO35 / GPIO49 / GPIO50 |
| Link mode | Autonegotiation advertising 100 Mbps full duplex only |

Change the example IP and MAC when using multiple units on one network.
The current Mac obtains `169.254.180.218` automatically on `en5`. Link resets
can remove that address temporarily while macOS repeats address acquisition.

## Build and flash

From the repository root, with Go and esptool 5 installed:

```sh
mkdir -p sandbox/poe-p4
go build -o sandbox/poe-p4/renvo ./cmd/renvo
sandbox/poe-p4/renvo -backend backends/esp32p4.rtg \
  -t esp32p4/riscv32 -o sandbox/poe-p4/tcp.elf \
  ./examples/device/poe_p4_tcp
esptool --chip esp32p4 elf2image --flash-size 16MB \
  -o sandbox/poe-p4/tcp.bin sandbox/poe-p4/tcp.elf
esptool --chip esp32p4 --port /dev/cu.usbmodem1101 \
  write-flash 0x10000 sandbox/poe-p4/tcp.bin
```

Use the actual serial port and esptool executable on your host. On this host,
esptool is available as `sandbox/esptool-venv/bin/python -m esptool`.
The USB console reports `RENVO POE-P4 TCP READY 169.254.180.4:4242` once the
link is ready. PHY/clock/link errors are reported repeatedly over USB.

Before the first application write, read the flash and inspect the partition
table. This unit has a 1 MiB factory application at `0x10000`, with SPIFFS at
`0x110000`; keep the application binary within that partition. The commands
above write only the application. They preserve bootloader, NVS and SPIFFS.

The original 16 MiB backup for this physical unit is
`sandbox/poe-p4/factory-flash.bin`, SHA-256:

```text
ece00d06ee60154e55b01f23af13ef171315d3ae4f0bb7a718f48206708f8c34
```

Keep that ignored file separately if the checkout may be cleaned. To restore
only the original application, extract bytes `0x10000:0x110000` from that
backup and flash the resulting 1 MiB file at `0x10000`.

## Verify from macOS

```sh
python3 examples/device/poe_p4_tcp/verify.py --interface en5
```

The script waits for the adapter's IPv4 address, binds the socket to that
interface, and verifies arbitrary random data over three connections. Each
connection transfers payloads of 1, 31, 512, 1024, 8192 and 65536 bytes, then
checks orderly EOF. It needs no packet-capture privileges. Binding matters:
macOS may otherwise choose a competing link-local route through Wi-Fi.

For a quick manual exchange using the current adapter address:

```sh
printf 'hello Renvo\n' | nc -s 169.254.180.218 -w 2 169.254.180.4 4242
ping -S 169.254.180.218 -c 3 169.254.180.4
```

To inspect the traffic, capture on the adapter in another terminal:

```sh
sudo tcpdump -i en5 -nn -s 0 -U -w /tmp/renvo-poe-p4.pcap
tcpdump -nn -r /tmp/renvo-poe-p4.pcap 'arp or tcp port 4242'
```

## Implementation and limits

`device/esp32p4.Ethernet` owns the EMAC and IP101. It uses eight receive
descriptors and one synchronous transmit descriptor. The example reserves
14,400 bytes at `0x4ff41000` exclusively for descriptors/buffers, above the
target's BSS, ROM scratch and stack. Both descriptors and buffers are accessed
through the uncached SRAM alias. Do not reuse this region for other drivers
or access it through the cached alias. Only the checked 128/256 KiB L2 cache
layouts are accepted.

The P4's 256-byte RX FIFO cannot hold a complete normal Ethernet frame.
DMA must stream from it: operation-mode RSF, bit 25, stays clear. Setting that
bit lets tiny packets work while larger packets overflow the FIFO.
Transmit operate-on-second-frame (OSF, bit 2) also stays clear: with one
self-chained TX descriptor, prefetch can read the same frame before ownership
writeback and transmit it twice.

`device/tcpip.Echo` implements ARP replies, ICMP echo, IPv4/TCP checksums,
one TCP connection, a maximum 512-byte segment, bounded retransmission,
duplicate/overlap handling and FIN shutdown. It is a diagnostic echo endpoint,
not a general socket API. It has no DHCP, routing, IPv6, VLAN support, IP
reassembly, TCP extensions, or concurrent clients. Idle connections expire
after 60 seconds. The single TIME-WAIT slot can be replaced by a fresh
connection from another peer port. A 10 Mbps-only peer is unsupported.

The main loop rewinds Renvo's scratch arena after synchronous transmission.
Long-lived endpoint state and copied retransmission data are allocated before
that mark. Applications using the endpoint need the same arena lifetime
discipline; do not retain temporary per-frame allocations across the rewind.

Host protocol tests:

```sh
go test ./device/tcpip ./device/esp32p4
./tools/check preflight
```

Verified on this physical unit on 2026-09-13: all three random-payload TCP
sessions passed, with 64 KiB echoes taking approximately 0.68 seconds each;
orderly TCP shutdown passed; three pings had no loss or duplicate replies.
The wire capture shows valid TCP checksums. Protocol tests passed, and
repository preflight completed in 46 seconds within its 60-second budget.

Hardware references:

- [M5Stack board pin map and specifications](https://docs.m5stack.com/en/unit/Unit_PoE-P4)
- [Espressif P4 EMAC register operations](https://github.com/espressif/esp-idf/blob/08e0d30a74ad0bfd5a34933142b80f45619ee410/components/esp_hal_emac/esp32p4/include/hal/emac_ll.h)
- [Espressif EMAC DMA descriptors](https://github.com/espressif/esp-idf/blob/08e0d30a74ad0bfd5a34933142b80f45619ee410/components/esp_hal_emac/include/hal/emac_hal.h)

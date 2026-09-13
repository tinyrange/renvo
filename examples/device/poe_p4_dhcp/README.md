# Unit PoE-P4: DHCP + TCP echo

This is the DHCP version of [the static-address example](../poe_p4_tcp/README.md).
It runs the same Renvo Ethernet driver and TCP echo service on port **4242**,
with an address obtained from your network's DHCP server.

The client uses the board's unique factory eFuse MAC and sends the DHCP
hostname **`renvo-poe-p4`**. On the unit used for bring-up, the MAC is
**`80:f1:b2:d1:4c:be`**. Find that MAC/hostname in your router's DHCP leases,
or read the USB serial console at 115200 baud. Every ten seconds it reports:

```text
DHCP TCP READY <leased-ip>:4242 mask <mask> gateway <router> lease <seconds> seconds ACKs <count>
```

The service becomes available only after the server's ACK and ARP conflict
probes. Removing the Ethernet cable immediately invalidates the local lease
once link polling detects the change; reconnecting starts a fresh DHCP
exchange. The application retries initial link setup if it boots without a
cable. Keep USB power connected when moving the cable unless the regular
network supplies PoE.

## Build and flash

From the repository root:

```sh
go build -o sandbox/poe-p4/renvo ./cmd/renvo
sandbox/poe-p4/renvo -backend backends/esp32p4.rtg \
  -t esp32p4/riscv32 -o sandbox/poe-p4/dhcp.elf \
  ./examples/device/poe_p4_dhcp
sandbox/esptool-venv/bin/python -m esptool --chip esp32p4 \
  elf2image --flash-size 16MB -o sandbox/poe-p4/dhcp.bin sandbox/poe-p4/dhcp.elf
sandbox/esptool-venv/bin/python -m esptool --chip esp32p4 \
  --port /dev/cu.usbmodem1101 write-flash 0x10000 sandbox/poe-p4/dhcp.bin
```

The same factory flash backup, 1 MiB application partition limit, and DMA
memory reservation described in the static example apply. This writes only
the application, preserving the original bootloader, NVS and SPIFFS.

## Test on a regular LAN

Connect the unit to the network, wait for its lease, then substitute the
reported IP below. Use the Mac interface connected to that network (`en0`
is commonly Wi-Fi); this can differ from the direct-cable adapter `en5`.

```sh
python3 examples/device/poe_p4_tcp/verify.py \
  --interface en0 --address <leased-ip>
nc <leased-ip> 4242
```

The existing verification script checks arbitrary random payloads through
64 KiB on three connections, plus orderly shutdown. The TCP implementation
is still a single-connection diagnostic echo service, not a general socket
library. The DHCP hostname is not an mDNS advertisement: use the leased IP
unless your router supplies DNS for DHCP hostnames.

## Isolated direct-cable DHCP fixture

With the board connected only to the Mac's USB Ethernet adapter:

```sh
python3 examples/device/poe_p4_dhcp/lab_server.py --interface en5
```

This temporary UDP fixture answers only the specified board MAC, is scoped
to the chosen interface, and stops after 90 seconds. It gives that one board
`169.254.180.44` with a 30-second lease, renewal at 12 seconds, and rebinding
at 24 seconds. The link-local address is solely an isolated test convenience
so the Mac needs no privileged interface reconfiguration; a regular DHCP
server should allocate addresses from its ordinary LAN subnet.

The fixture does not answer the Mac's own DHCP requests. After flashing,
macOS can take several seconds to restore its automatic link-local address;
DHCP retransmissions handle this delay. Adjust `--server`, `--address` and
`--mac` when the test adapter/board differs. Do not run this fixture on your
regular LAN.

```sh
python3 examples/device/poe_p4_tcp/verify.py \
  --interface en5 --address 169.254.180.44
```

`--skip-renewals` withholds T1 ACKs so the client must rebind at T2.
`--expire` withholds all replies after the first ACK, to exercise expiry.
The firmware removes an expired address and returns to discovery; it never
falls back to the static example's address.

## DHCP behavior

`device/tcpip.DHCP` supports DISCOVER/OFFER/REQUEST/ACK, NAK restart,
exponential acquisition retry, unicast renewal (with ARP for the server or
gateway), broadcast rebinding, expiry and infinite leases. It reads subnet
mask, router, first DNS server, lease duration, T1 and T2, including BOOTP
option-overload fields. DNS configuration is recorded; no DNS resolver is
provided. It validates checksums, transaction ID, client MAC, server identity,
option lengths and lease configuration before using a response.

Before using a new address, it sends three randomized ARP probes and two
announcements. Conflicts cause DECLINE, removal of the address and backoff.
Long leases use second counters rather than overflowing millisecond duration
arithmetic. Scratch arena storage is reclaimed by the main polling loop.

There is no persistent lease cache, DHCPv6, DHCP authentication or fragmented
IP reassembly. Split/repeated singleton options are rejected; option
concatenation is not implemented. The Ethernet driver currently advertises
only 100 Mbps full duplex.

```sh
go test ./device/tcpip ./device/esp32p4
./tools/check preflight
```

Protocol references: [DHCPv4](https://www.rfc-editor.org/rfc/rfc2131.html),
[DHCP options](https://www.rfc-editor.org/rfc/rfc2132.html), and
[IPv4 address conflict detection](https://www.rfc-editor.org/rfc/rfc5227.html).

Hardware validation on 2026-09-13 covered discovery/request/ACK, ARP probing,
repeated unicast renewal, broadcast rebinding with renewal replies withheld,
expiry/reacquisition, and three TCP sessions through 64 KiB random payloads.
The protocol tests cover NAK, conflicts/DECLINE, malformed replies, option
overload, long/infinite leases and clock wrapping. Repository preflight passed
in 47 seconds within its 60-second budget.

Regular-LAN validation also passed: the router assigned `192.168.1.134/24`
with gateway `192.168.1.1` and an 86,400-second lease. From the Mac's Wi-Fi
address `192.168.1.140`, all three TCP sessions passed (including 64 KiB
payloads in approximately 1.6 seconds and orderly shutdown). Three pings had
no loss. These are observed DHCP addresses, not fixed firmware settings.

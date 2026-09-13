# Unit PoE-P4: DHCP web server

A standalone Renvo HTTP server on Ethernet port **80**, using the board's
factory MAC and DHCP hostname `renvo-poe-p4`. USB serial at 115200 baud prints
the URL every ten seconds after DHCP completes.

- `/`: status page with IP, MAC, subnet, gateway, uptime and DHCP information.
- `/status.json`: the same information as JSON.
- `/healthz`: `ok` followed by a newline.

GET and HEAD are supported. Four fixed connection slots accommodate overlapping
browser requests. Responses span TCP packets and support retransmission;
each response closes its connection. Header storage is limited to 2048 bytes,
with a five-second request timeout. This small example has no TLS, request
bodies, keep-alive, pipelining or general socket API.

Build from the repository root:

```sh
go build -o sandbox/poe-p4/renvo ./cmd/renvo
sandbox/poe-p4/renvo -backend backends/esp32p4.rtg \
  -t esp32p4/riscv32 -o sandbox/poe-p4/http.elf \
  ./examples/device/poe_p4_http
sandbox/esptool-venv/bin/python -m esptool --chip esp32p4 \
  elf2image --flash-size 16MB -o sandbox/poe-p4/http.bin sandbox/poe-p4/http.elf
```

Check the binary is at most **1 MiB** before writing the factory application
partition. The [original bring-up instructions](../poe_p4_tcp/README.md)
describe the flash backup and reserved DMA memory. Flash only the application:

```sh
sandbox/esptool-venv/bin/python -m esptool --chip esp32p4 \
  --port /dev/cu.usbmodem1101 write-flash 0x10000 sandbox/poe-p4/http.bin
```

Connect to your regular LAN, read the leased address from serial, then open
`http://<leased-ip>/` or run:

```sh
curl http://<leased-ip>/status.json
python3 examples/device/poe_p4_http/verify.py <leased-ip>
go test ./device/tcpip ./device/esp32p4
./tools/check preflight
```

DHCP acquisition, conflict detection, renewal and link handling match the
[DHCP echo example](../poe_p4_dhcp/README.md). The static and DHCP echo programs
remain separate examples; this firmware serves HTTP instead of port 4242 echo.

Validated on hardware on 2026-09-13 at `192.168.1.134`: page and JSON retrieval,
health check, HEAD, 404/405/204 responses, a split request, and 40 page requests
with four concurrent clients. The image was 967,392 bytes, within the factory
1 MiB application partition. Device tests, focused compiler regressions,
preflight and the native compiler performance gate passed.

This example also exposed a compiler layout bug for arrays containing
later-declared structs. The compiler now resolves value layout dependencies
before calculating array strides and containing field offsets; the regression
is `backend/tests/forward_array_struct_layout.go`.

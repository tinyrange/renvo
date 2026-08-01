#!/usr/bin/env bash

set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
	echo "usage: $0 ELF [PORT]" >&2
	exit 2
fi

elf=$1
port=${2:-/dev/ttyACM0}
image=${elf%.elf}.bin
esptool=${ESPTOOL:-esptool.py}

"$esptool" --chip esp32c6 elf2image \
	--flash_mode dio --flash_freq 80m --flash_size 4MB \
	-o "$image" "$elf"

# The ESP32-C6 native USB CDC endpoint needs longer after RTS is released than
# esptool 4.x's generic hard reset provides. Leave the flasher stub alive, then
# reset with the USB-aware timing. Keeping the port open during the settle time
# prevents its close transition from putting the chip back in the ROM loader.
"$esptool" --chip esp32c6 --port "$port" --after no_reset_stub \
	write_flash 0x10000 "$image"

M5NANOC6_PORT=$port python3 - <<'PY'
import os
import sys
import time

import serial

device = serial.Serial(os.environ["M5NANOC6_PORT"], 115200, timeout=0.1)
device.dtr = False
device.rts = True
time.sleep(0.2)
device.rts = False
deadline = time.monotonic() + 1.0
while time.monotonic() < deadline:
	data = device.read(4096)
	if data:
		sys.stdout.buffer.write(data)
		sys.stdout.buffer.flush()
device.close()
PY

echo "Flashed $image and started the application"

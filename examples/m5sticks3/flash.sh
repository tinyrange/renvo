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

"$esptool" --chip esp32s3 elf2image \
	--flash-mode dio --flash-freq 80m --flash-size 8MB \
	-o "$image" "$elf"

"$esptool" --chip esp32s3 --port "$port" --after watchdog-reset \
	write-flash 0x10000 "$image"

echo "Flashed $image to the factory application partition and started it"

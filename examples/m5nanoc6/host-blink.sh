#!/usr/bin/env bash

set -euo pipefail

if [[ $# -gt 3 ]]; then
	echo "usage: $0 [CYCLES [HALF-PERIOD-MS [USB-DEVICE]]]" >&2
	exit 2
fi

cycles=${1:-6}
half_period_ms=${2:-250}
usb_device=${3:-}
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)

if [[ -n ${RENVOFLASH:-} ]]; then
	command=("$RENVOFLASH")
else
	command=(go run ./cmd/renvoflash)
fi

args=(--jtag-blink "$cycles" "$half_period_ms")
if [[ -n $usb_device ]]; then
	args+=("$usb_device")
fi

cd "$root"
exec "${command[@]}" "${args[@]}"

#!/bin/sh
set -eu

example_root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repository_root=$(CDPATH= cd -- "$example_root/../.." && pwd)
emulator_root=${RENVO_EMU_ROOT:-"$repository_root/../renvo_emu"}
temporary=$(mktemp -d)
trap 'rm -rf "$temporary"' EXIT HUP INT TERM

renvo=${RENVO:-"$temporary/renvo"}
if [ -z "${RENVO:-}" ]; then
	(cd "$repository_root" && go build -o "$renvo" ./cmd/renvo)
fi

emulator=${RENVO_EMU:-"$emulator_root/target/debug/remu"}
if [ ! -x "$emulator" ]; then
	(cd "$emulator_root" && cargo build --quiet -p remu-cli --bin remu)
fi

"$renvo" -backend "$example_root/esp32s3.rtg" -t esp32s3/xtensa_lx7 \
	-o "$temporary/usb-cdc.elf" "$example_root/usb_cdc"

"$emulator" run --target esp32s3 --elf "$temporary/usb-cdc.elf" \
	--max-instructions 3000000 --result "$temporary/result.json"

compact=$(tr -d '[:space:]' < "$temporary/result.json")
case "$compact" in
	*'"usb":[80,65,83,83,10]'*) ;;
	*)
		cat "$temporary/result.json" >&2
		echo "renvo_emu did not enumerate DWC2 CDC and receive PASS" >&2
		exit 1
		;;
esac

echo "PASS: ESP32-S3 DWC2 enumerated and transferred CDC data"

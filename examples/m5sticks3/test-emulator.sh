#!/bin/sh
set -eu

example_root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repository_root=$(CDPATH= cd -- "$example_root/../.." && pwd)
emulator_root=${RENVO_EMU_ROOT:-"$repository_root/../renvo_emu"}

if [ -z "${RENVO_EMU:-}" ] && [ ! -f "$emulator_root/Cargo.toml" ]; then
	echo "renvo_emu not found at $emulator_root; set RENVO_EMU_ROOT" >&2
	exit 2
fi

temporary=$(mktemp -d)
trap 'rm -rf "$temporary"' EXIT HUP INT TERM

if [ -n "${RENVO:-}" ]; then
	renvo=$RENVO
else
	renvo="$temporary/renvo"
	(cd "$repository_root" && go build -o "$renvo" ./cmd/renvo)
fi

if [ -n "${RENVO_EMU:-}" ]; then
	emulator=$RENVO_EMU
else
	emulator="$emulator_root/target/debug/remu"
	if [ ! -x "$emulator" ]; then
		(cd "$emulator_root" && cargo build --quiet -p remu-cli --bin remu)
	fi
fi

"$renvo" \
	-backend "$repository_root/backends/esp32s3.rtg" \
	-t esp32s3/xtensa_lx7 \
	-o "$temporary/suite.elf" \
	"$repository_root/frontend_tests/single_file_microcontroller"

"$emulator" run \
	--target esp32s3 \
	--elf "$temporary/suite.elf" \
	--max-instructions 1000000 \
	--result "$temporary/result.json"

compact=$(tr -d '[:space:]' < "$temporary/result.json")
case "$compact" in
	*'"usb":[80,65,83,83,10]'*) ;;
	*)
		cat "$temporary/result.json" >&2
		echo "renvo_emu did not receive PASS through USB Serial/JTAG" >&2
		exit 1
		;;
esac

instructions=$(sed -n 's/^[[:space:]]*"instructions": \([0-9][0-9]*\),/\1/p' "$temporary/result.json")
echo "PASS: frontend microcontroller suite wrote PASS after at most $instructions instructions"

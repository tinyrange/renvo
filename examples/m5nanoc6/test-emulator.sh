#!/bin/sh
set -eu

example_root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repository_root=$(CDPATH= cd -- "$example_root/../.." && pwd)
emulator_root=${RENVO_EMU_ROOT:-"$repository_root/../renvo_emu"}

if [ ! -f "$emulator_root/Cargo.toml" ]; then
	echo "renvo_emu not found at $emulator_root; set RENVO_EMU_ROOT" >&2
	exit 2
fi

temporary=$(mktemp -d)
trap 'rm -rf "$temporary"' EXIT HUP INT TERM

if [ -n "${RENVO:-}" ]; then
	renvo=$RENVO
	if [ ! -x "$renvo" ]; then
		echo "RENVO is not executable: $renvo" >&2
		exit 2
	fi
else
	renvo="$temporary/renvo"
	(cd "$repository_root" && go build -o "$renvo" ./cmd/renvo)
fi

if [ -n "${RENVO_EMU:-}" ]; then
	emulator=$RENVO_EMU
	if [ ! -x "$emulator" ]; then
		echo "RENVO_EMU is not executable: $emulator" >&2
		exit 2
	fi
else
	emulator="$emulator_root/target/debug/renvo"
	if [ ! -x "$emulator" ]; then
		(cd "$emulator_root" && cargo build --quiet -p renvo-cli --bin renvo)
	fi
fi

"$renvo" \
	-backend "$example_root/esp32c6.rtg" \
	-t esp32c6/riscv32 \
	-o "$temporary/oracle.elf" \
	"$example_root/oracle"

"$emulator" run \
	--target esp32c6 \
	--elf "$temporary/oracle.elf" \
	--max-instructions 500000 \
	--stop-signal board.esp32c6.chip_gpio.pin7=rising \
	--result "$temporary/result.json"

if ! grep -Fq '"Signal": "board.esp32c6.chip_gpio.pin7"' "$temporary/result.json"; then
	cat "$temporary/result.json" >&2
	echo "renvo_emu did not observe the NanoC6 blue LED rising edge" >&2
	exit 1
fi

instructions=$(sed -n 's/^[[:space:]]*"instructions": \([0-9][0-9]*\),/\1/p' "$temporary/result.json")
echo "PASS: renvo_emu observed GPIO7 rising after $instructions instructions"

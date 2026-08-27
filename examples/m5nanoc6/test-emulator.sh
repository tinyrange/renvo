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
	emulator="$emulator_root/target/debug/remu"
	if [ ! -x "$emulator" ]; then
		(cd "$emulator_root" && cargo build --quiet -p remu-cli --bin remu)
	fi
fi

run_target() {
	target=$1
	definition=$2
	name=$(printf '%s' "$target" | tr / -)
	image="$temporary/$name.elf"
	result="$temporary/$name.json"

	"$renvo" \
		-backend "$repository_root/$definition" \
		-t "$target" -tags m5nanoc6 \
		-o "$image" \
		"$example_root/oracle"

	"$emulator" run \
		--target esp32c6 \
		--elf "$image" \
		--max-instructions 2000000 \
		--stop-signal board.esp32c6.chip_gpio.pin7=rising \
		--result "$result"

	if ! grep -Fq '"Signal": "board.esp32c6.chip_gpio.pin7"' "$result"; then
		cat "$result" >&2
		echo "renvo_emu did not observe the NanoC6 blue LED rising edge for $target" >&2
		exit 1
	fi

	instructions=$(sed -n 's/^[[:space:]]*"instructions": \([0-9][0-9]*\),/\1/p' "$result")
	echo "PASS: $target raised GPIO7 after $instructions instructions"
}

run_target esp32c6/riscv32 backends/esp32c6.rtg
run_target esp32c6-jtag/riscv32 backends/esp32c6_jtag.rtg

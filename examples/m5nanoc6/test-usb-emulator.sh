#!/bin/sh
set -eu

example_root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repository_root=$(CDPATH= cd -- "$example_root/../.." && pwd)
emulator_root=${RENVO_EMU_ROOT:-"$repository_root/local/renvo_emu"}
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

"$renvo" -backend "$example_root/esp32c6.rtg" -t esp32c6/riscv32 \
	-o "$temporary/usb-low-speed.elf" "$example_root/usb_low_speed"

set -- run --target esp32c6 --elf "$temporary/usb-low-speed.elf" \
	--max-instructions 250000 --result "$temporary/result.json"
if [ -n "${ESPTOOL:-}" ]; then
	"$ESPTOOL" --chip esp32c6 elf2image \
		--flash-mode dio --flash-freq 80m --flash-size 4MB \
		-o "$temporary/usb-low-speed.bin" "$temporary/usb-low-speed.elf"
	set -- "$@" --esp-app-image "$temporary/usb-low-speed.bin"
fi
"$emulator" "$@"

compact=$(tr -d '[:space:]' < "$temporary/result.json")
case "$compact" in
	*'"usb":[67,54,32,80,65,83,83,10]'*) ;;
	*)
		cat "$temporary/result.json" >&2
		echo "renvo_emu rejected the ESP32-C6 raw-PHY low-speed packet" >&2
		exit 1
		;;
esac

"$renvo" -backend "$example_root/esp32c6.rtg" -t esp32c6/riscv32 \
	-o "$temporary/usb-recovery.elf" "$example_root/usb_recovery"
"$emulator" run --target esp32c6 --elf "$temporary/usb-recovery.elf" \
	--max-instructions 300000 \
	--stop-signal board.esp32c6.chip_gpio.pin7=rising \
	--result "$temporary/recovery-result.json"
if ! grep -Fq '"Signal": "board.esp32c6.chip_gpio.pin7"' \
	"$temporary/recovery-result.json"; then
	cat "$temporary/recovery-result.json" >&2
	echo "renvo_emu did not reboot into the C6 USB recovery window" >&2
	exit 1
fi

echo "PASS: ESP32-C6 USB packet path and watchdog recovery window"

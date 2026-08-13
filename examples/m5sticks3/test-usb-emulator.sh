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

if [ "$#" -eq 0 ]; then
	set -- cdc hid cdc_ethernet midi audio webusb mtp adb msc
fi

for profile do
	case "$profile" in
		cdc) expected='"usb":[80,65,83,83,10]'; required= ;;
		hid) expected='72,73,68,32,80,65,83,83,10'; required= ;;
		cdc_ethernet) expected='"usb":[161,0,1,0,0,0,0,0'; required='69,67,77,32,80,65,83,83,10' ;;
		midi) expected='"usb":[9,144,60,127]'; required= ;;
		audio) expected='65,85,68,73,79,32,80,65,83,83,10'; required='90,165],"trace_digest"' ;;
		webusb) expected='87,69,66,85,83,66,32,80,65,83,83,10'; required= ;;
		mtp) expected='"usb":[21,0,0,0,3,0,1,32,7,0,0,0'; required='12,0,0,0,4,0,2,64,7,0,0,0' ;;
		adb) expected='"usb":[67,78,88,78,1,0,0,1,0,16,0,0,9,0,0,0'; required='65,68,66,32,80,65,83,83,10' ;;
		msc) expected='82,69,78,86,79,32,32,32,85,83,66,32,66,76,79,67,75'; required='85,83,66,83,7,0,0,0' ;;
		*)
			echo "unknown USB profile: $profile" >&2
			exit 2
			;;
	esac

	"$renvo" -backend "$example_root/esp32s3.rtg" -t esp32s3/xtensa_lx7 \
		-o "$temporary/usb-$profile.elf" "$example_root/usb_$profile"
	"$emulator" run --target esp32s3 --elf "$temporary/usb-$profile.elf" \
		--max-instructions 800000 --result "$temporary/$profile.json"

	compact=$(tr -d '[:space:]' < "$temporary/$profile.json")
	case "$compact" in
		*"$expected"*"$required"*) ;;
		*)
			cat "$temporary/$profile.json" >&2
			echo "renvo_emu did not complete the $profile USB transaction" >&2
			exit 1
			;;
	esac
	echo "PASS: $profile"
done

#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
probe="$repo_root/sandbox/esp32c6-fs-romload.elf"
image="${probe%.elf}.bin"
usb_trace=/tmp/renvo-esp32c6-fs-romload-usbmon.txt
gcc_path=$(find "$HOME/.espressif/tools/riscv32-esp-elf" -type f -name riscv32-esp-elf-gcc | sort -V | tail -1)
esptool_path=${ESPTOOL:-$HOME/.espressif/python_env/idf6.2_py3.14_env/bin/esptool}
usbmon_pid=

cleanup() {
	if [[ -n "$usbmon_pid" ]]; then
		wait "$usbmon_pid" 2>/dev/null || true
	fi
	sudo systemctl start ModemManager.service >/dev/null 2>&1 || true
}
trap cleanup EXIT

"$gcc_path" -nostdlib -march=rv32imac_zicsr -mabi=ilp32 \
	-DSTOP_AFTER_FIRST_C=${STOP_AFTER_FIRST:-1} \
	-Wl,-Ttext=0x40800000,--no-relax \
	-Wa,--defsym,DIAGNOSTIC=0,--defsym,RX_CENTER=0 \
	-Wa,--defsym,STOP_AFTER_STATUS=${STOP_AFTER_STATUS:-0} \
	-Wa,--defsym,TRACE_STAGE=0,--defsym,FIRST_ZLP=0,--defsym,FIRST_FULL=${FIRST_FULL:-0} \
	-Wa,--defsym,FIRST_QUAL=${FIRST_QUAL:-0},--defsym,FIRST_CONFIG=${FIRST_CONFIG:-0} \
	-Wa,--defsym,FIRST_TWO=${FIRST_TWO:-0},--defsym,TEST_UNSTUFFED_TWO=0 \
	-Wa,--defsym,GPIO_TX=${GPIO_TX:-0},--defsym,GPIO_ATTACH_ONLY=${GPIO_ATTACH_ONLY:-0} \
	-Wa,--defsym,GPIO_ACK=${GPIO_ACK:-0} \
	-Wa,--defsym,GPIO_CELL_NOPS=${GPIO_CELL_NOPS:-9},--defsym,RESET_FIXED=${RESET_FIXED:-1} \
	-Wa,--defsym,HYBRID_TX=${HYBRID_TX:-0},--defsym,RESET_DURING_TX=${RESET_DURING_TX:-0} \
	-Wa,--defsym,RESET_FIXED_LATE=${RESET_FIXED_LATE:-0} \
	-Wa,--defsym,MMIO_TIMING=${MMIO_TIMING:-0} \
	-Wa,--defsym,TEST_EXACT=${TEST_EXACT:-0} \
	-Wa,--defsym,TEST_PATTERN=${TEST_PATTERN:-0} \
	-Wa,--defsym,EOP_RELEASE_J=${EOP_RELEASE_J:-0} \
	-Wa,--defsym,EOP_EXTRA_SE0=${EOP_EXTRA_SE0:-0} \
	-Wa,--defsym,TX_PATTERN_NOPS=${TX_PATTERN_NOPS:-1} \
	-Wa,--defsym,TX_PATTERN_PHASE=${TX_PATTERN_PHASE:-0} \
	-Wa,--defsym,FINAL2_PHASE=${FINAL2_PHASE:-0} \
	-Wa,--defsym,FULL_PHASE=${FULL_PHASE:-0} \
	-Wa,--defsym,ACK_PHASE=${ACK_PHASE:-0} \
	-Wa,--defsym,ACK_RELEASE=${ACK_RELEASE:-0} \
	-Wa,--defsym,ACK_ROTATE=${ACK_ROTATE:-0} \
	-Wa,--defsym,TX_CPU160=${TX_CPU160:-0} \
	-Wa,--defsym,TX_RELEASE_NOPS=${TX_RELEASE_NOPS:-0} \
	-Wa,--defsym,REL8=${REL8:-0},--defsym,REL18=${REL18:-2} \
	-Wa,--defsym,RELQUAL=${RELQUAL:-0},--defsym,RELCONFIG=${RELCONFIG:-0} \
	-Wa,--defsym,TX_PRIME_WRITES=${TX_PRIME_WRITES:-0} \
	-Wa,--defsym,TEST_SCHEDULED=${TEST_SCHEDULED:-0} \
	-Wa,--defsym,SCHEDULE_CPU160=${SCHEDULE_CPU160:-0} \
	-Wa,--defsym,SCHEDULE_CYCLES=${SCHEDULE_CYCLES:-10},--defsym,SCHEDULE_EXTRA=${SCHEDULE_EXTRA:-0} \
	-Wa,--defsym,SCHEDULE_PERIOD=${SCHEDULE_PERIOD:-3},--defsym,SCHEDULE_PHASE=${SCHEDULE_PHASE:-0} \
	-Wa,--defsym,SCHEDULE_START=${SCHEDULE_START:-16} \
	-Wa,--defsym,TX_DITHER=${TX_DITHER:-0} \
	-Wa,--defsym,TEST_CELL_NOPS=${TEST_CELL_NOPS:-9} \
	-Wa,--defsym,GPIO_DRIVE_TEST=${GPIO_DRIVE_TEST:-0} \
	-Wa,--defsym,MMIO_GAP=${MMIO_GAP:-0} \
	-Wa,--defsym,MMIO_PATTERN=${MMIO_PATTERN:-0} \
	-Wa,--defsym,MMIO_CPU80=${MMIO_CPU80:-0} \
	-Wa,--defsym,MMIO_CPU160=${MMIO_CPU160:-0} \
	-Wa,--defsym,TEST_ONE=0,--defsym,TX_PERIOD=${TX_PERIOD:-5} \
	-Wa,--defsym,CAPTURE_AFTER_ACK=${CAPTURE_AFTER_ACK:-0},--defsym,TX_PHASE=${TX_PHASE:-0} \
	-Wa,--defsym,RESPONSE_OFFSET=${RESPONSE_OFFSET:-46} \
	-Wa,--defsym,WATCHDOG_TICKS=${WATCHDOG_TICKS:-60000} \
	-o "$probe" "$repo_root/sandbox/esp32c6-fs-nak-sie.S"
"$esptool_path" --chip esp32c6 elf2image --ram-only-header --use-segments \
	-o "$image" "$probe" >/dev/null

if [[ ${BUILD_ONLY:-0} == 1 ]]; then
	exit 0
fi

sudo systemctl stop ModemManager.service >/dev/null 2>&1 || true
bus=$(lsusb -d 303a:1001 | awk 'NR == 1 { print int($2) }')
sudo timeout "${TRACE_SECONDS:-7}s" cat "/sys/kernel/debug/usb/usbmon/${bus}u" >"$usb_trace" &
usbmon_pid=$!
sleep 0.2
loaded=0
load_log=/tmp/renvo-esp32c6-fs-romload-esptool.txt
for _ in 1 2 3; do
	if timeout 8s "$esptool_path" --chip esp32c6 --port /dev/ttyACM0 \
		--before default-reset --after no-reset --no-stub load-ram "$image" \
		2>&1 | tee "$load_log"; then
		loaded=1
		break
	fi
	# The image deliberately disconnects the ROM CDC endpoint. pySerial may
	# report that detach while esptool is closing an otherwise successful load.
	if rg -q 'Loaded [0-9]+ segments.*executing at' "$load_log"; then
		loaded=1
		break
	fi
	usb_port=/sys/bus/usb/devices/usb3/3-0:1.0/usb3-port1
	echo 1 | sudo tee "$usb_port/disable" >/dev/null
	sleep 0.2
	echo 0 | sudo tee "$usb_port/disable" >/dev/null
	for _ in $(seq 1 50); do
		[[ -e /dev/ttyACM0 ]] && break
		sleep 0.1
	done
done
if [[ $loaded == 0 ]]; then
	exit 1
fi
wait "$usbmon_pid" 2>/dev/null || true
usbmon_pid=

if [[ ${PORT_RECOVER:-0} == 1 ]]; then
	usb_port=/sys/bus/usb/devices/usb3/3-0:1.0/usb3-port1
	echo 1 | sudo tee "$usb_port/disable" >/dev/null
	sleep 0.2
	echo 0 | sudo tee "$usb_port/disable" >/dev/null
	sleep 1
fi

rg '12011001|addec600|S Ci:.* s 80 06|C Ci:' "$usb_trace" | tail -100 || true
if lsusb -d dead:00c6; then
	exit 0
fi
for _ in $(seq 1 100); do
	if [[ -e /dev/ttyACM0 ]]; then
		break
	fi
	sleep 0.1
done
"$esptool_path" --chip esp32c6 --port /dev/ttyACM0 --before default-reset \
	--after hard-reset --no-stub read-mem 0x600b1020

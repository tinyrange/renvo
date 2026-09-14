# Chain DualKey switches

This example runs directly on the DualKey's ESP32-S3. It debounces the two
active-low switches independently for 20 ms and prints press/release events
and press counters through USB Serial/JTAG. Released keys are blue; held keys
are green. Both keys can be held together.

The [M5Stack pin map](https://docs.m5stack.com/en/chain/Chain_DualKey) assigns
Key1 (farther from the lanyard hole) to GPIO0, Key2 to GPIO17, and LED data to
GPIO21. GPIO40 enables the LED supply **low**, as implemented by the
[factory driver](https://github.com/m5stack/M5DualKey-UserDemo/blob/main/components/esp_duo/esp_duo.c).
The LED chain reaches Key2 before Key1. GPIO7 and GPIO8 control the power
switch sensing and must not be driven high; this example leaves them alone.

Build with a current Renvo compiler:

```sh
renvo -backend backends/esp32s3.rtg -t esp32s3/xtensa_lx7 \
  -o sandbox/dualkey-switches.elf ./examples/device/dualkey_switches
python -m esptool --chip esp32s3 elf2image --flash-size 8MB \
  -o sandbox/dualkey-switches.bin sandbox/dualkey-switches.elf
```

Before replacing factory firmware, back up the full 8 MB flash and inspect its
partition table. The tested unit has its factory application at `0x20000`,
with a `0x380000` byte capacity. Preserve the bootloader, partition table, and
other partitions. With that partition layout confirmed and the binary within
the application capacity:

```sh
python -m esptool --chip esp32s3 --port /dev/cu.usbmodem101 \
  --after watchdog-reset write-flash 0x20000 sandbox/dualkey-switches.bin
```

Use the serial port reported by your host. To enter download mode manually,
put the switch in the middle, unplug USB, hold Key1, reconnect USB, then release
Key1. A watchdog reset after flashing successfully starts the application;
the default hard reset can leave this board in the ROM downloader.

This example retains USB Serial/JTAG; it does not enumerate as USB HID.

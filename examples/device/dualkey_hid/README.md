# Chain DualKey USB HID

This standalone Renvo firmware enumerates the ESP32-S3 as **Renvo DualKey**,
a full-speed USB HID keyboard. The physical mapping was confirmed on the
attached board:

| Button | GPIO | Key |
| --- | --- | --- |
| Left (Key2) | GPIO17 | `a` |
| Right (Key1) | GPIO0 | `b` |

Each switch has independent 20 ms debounce. Its own LED is blue when released
and green while held. Eight-byte keyboard reports include both keys when held
together and clear released keys. Holding a key uses the host's normal repeat
behavior; Shift and Caps Lock affect case as on an ordinary keyboard.
The implementation uses the S3's DWC2 USB OTG peripheral
directly, without ESP-IDF or TinyUSB linked into the firmware.

## Build and flash

Follow the backup and partition checks in
[the switch example](../dualkey_switches/README.md). Build this example instead:

```sh
renvo -backend backends/esp32s3.rtg -t esp32s3/xtensa_lx7 \
  -o sandbox/dualkey-hid.elf ./examples/device/dualkey_hid
python -m esptool --chip esp32s3 elf2image --flash-size 8MB \
  -o sandbox/dualkey-hid.bin sandbox/dualkey-hid.elf
python -m esptool --chip esp32s3 --port /dev/cu.usbmodem101 \
  --after watchdog-reset write-flash 0x20000 sandbox/dualkey-hid.bin
```

The example reuses the attached DualKey's factory VID/PID (`303a:8000`) for
local development. These are not identifiers assigned to Renvo; use suitable
identifiers and a unique serial number for distributed firmware.

## Recovery

The application waits five seconds at startup before USB switches from
Serial/JTAG to HID. The serial port then disappears because both controllers
share the internal PHY. Hold both keys for five seconds to disconnect HID and
return the PHY to Serial/JTAG; both LEDs become purple. USB flashing is then
available again, although console output may require a reboot.

For guaranteed ROM download recovery, put the power switch in the middle,
unplug USB, hold Key1 (farther from the lanyard hole), reconnect USB, and release
Key1. No working application is required for that recovery path.

## Host verification

On macOS, `ioreg -r -c IOHIDDevice -l -w 0` shows the product and Keyboard-page
elements. To test typing, focus a text field and press left then right: `ab`.
USB keyboard enumeration and repeated `ab` typing were verified on the attached
DualKey and macOS host, along with the return-to-flashing gesture.
With Python `hidapi` installed and keyboard input access available:

```python
import hid

device = hid.device()
device.open(0x303a, 0x8000, "DualKey-dev")
try:
    while True:
        report = device.read(8, 1000)
        if report:
            print(report)
finally:
    device.close()
```

Keyboard usage `0x04` is A and `0x05` is B; the first two bytes are zero and
the remaining six bytes contain held key usages. An all-zero report releases
both keys. The driver is polling-only and supports one keyboard or gamepad
interface, not composite CDC/HID or remote wakeup. Its default gamepad mode
remains available by omitting `Keyboard: true` from the configuration.

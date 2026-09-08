# Unit RollerCAN over I2C

`renvo.dev/device/motor/rollercan` implements the RollerCAN application's
complete I2C register interface, including the firmware v3 extensions. It works
with `i2c.New(board.Grove)` on the NanoC6 (GPIO2 SDA, GPIO1 SCL, 100 kHz).
Connect the unit's Grove port to the NanoC6 Grove port using the four-wire cable.
The default seven-bit address is `0x64`.

```go
motor := rollercan.New(i2c.New(board.Grove))
speed, err := motor.Speed() // Hundredths of RPM; does not enable the motor.
```

`New` and `NewAt` perform no transactions. Set output off, configure the mode,
current limit and target, then explicitly enable output. Every bus operation
returns an error; a successful write means the bus acknowledged it, not that
the motor reached its target. Check `Status` and `Faults` as well as readback.
Serialize operations if several callers share a device.

## Register coverage

| Registers | API |
| --- | --- |
| `00`, `01` | Output enable and speed/position/current/encoder mode |
| `0A`, `0B`, `0C`, `0D`, `0E`, `0F` | Position protection, clear stall, status, fault mask, button mode switching, stall protection |
| `10`, `11`, `12` | CAN ID, CAN baud rate, RGB brightness |
| `20–23` | Position loop current limit |
| `30–33` | RGB channels (BGR on wire) and system/user RGB mode |
| `34–37`, `38–3B`, `3C–3F` | Supply voltage, internal temperature, read/write dial count |
| `40–43`, `50–53`, `60–63`, `70–7B` | Speed target, current limit, actual speed, PID |
| `80–83`, `90–93`, `A0–AB` | Position target, actual position, PID |
| `88–8D`, `98–9D` | Extended target and actual turns/angle snapshots (v3+) |
| `B0–B3`, `C0–C3` | Target and actual phase current |
| `E0–EB`, `F4` | STM32 UID and product type (v3+) |
| `F0` | Explicit configuration save to flash |
| `F1`, `F2`, `F3` | Start encoder calibration, apply/save offset, busy status (v3+) |
| `FD` | Application reset command used by firmware updaters |
| `FE`, `FF` | Firmware version, read/change I2C address |

All supported read/write registers have both getters and setters. Extended
position fields are transferred together so turns and angle describe the same
snapshot. No reserved registers are written. The CAN interface's ID/baud settings
are accessible through I2C; CAN packet transport and CAN-to-I2C forwarding are
separate protocols. The bootloader's firmware upload protocol is also separate:
`ResetForBootloader` exposes the application reset register only, and does not
perform its bus-low entry handshake or flash a firmware image.

## Values and persistence

The API uses exact integer wire units:

- Speed: 0.01 RPM; `1000` means 10 RPM.
- Position: 0.01 degree; `36000` means one revolution.
- Current: 0.01 mA of motor phase current; `10000` means 100 mA.
- Voltage: 0.01 V; temperature: whole degrees Celsius.
- PID: unsigned P/D multiplied by 100000 and I multiplied by 10000000.
- Extended position: signed whole turns plus an angle from 0 through 35999.
  Minus one degree is `ExtendedPosition{Turns: -1, Angle: 35900}`.

Targets outside ±2100000000 and currents outside ±120000 are rejected before
I2C. Current-limit setters accept the protocol's signed range; current firmware
uses its magnitude. The unit's custom PID preset must be selected on the unit
for custom gains to take effect; reads report the active preset's gains.

`SaveConfig` and `SaveEncoderCalibration` explicitly write flash.
`SetI2CAddress` also persists immediately and accepts `0x08..0x77`, the range
accepted by current firmware. Each allows 100 ms to settle after a successful
write. Address changes update the driver's address only after acknowledgment;
if the bus fails during readdressing, check both addresses before retrying.
Flash operations belong in configuration workflows, not the control loop.

V3 extensions check `FirmwareVersion` first and return `ErrFirmware` for older
units. Legacy firmware may return the previous response for unknown registers;
blindly reading new registers can therefore produce plausible but false data.
Encoder calibration can move the motor. Poll `CalibrationBusy` until false
before explicitly applying/saving the result.

## NanoC6 example

The runnable example is [`examples/device/rollercan`](../../../examples/device/rollercan/main.go).
It starts with output disabled, enables both protections, and sets 10 RPM with
a 100 mA phase-current limit. Hold the NanoC6 button to run; release to disable
drive. A button held during boot must first be released. Blue means idle,
green means a run request, and red means an error. Telemetry prints every
roughly half-second. Configuration is not saved to flash.

```sh
go build -o sandbox/renvo ./cmd/renvo
sandbox/renvo \
  -backend backends/esp32c6.rtg \
  -t esp32c6/riscv32 -tags m5nanoc6 \
  -o sandbox/m5nanoc6-rollercan.elf \
  ./examples/device/rollercan
./examples/m5nanoc6/flash.sh sandbox/m5nanoc6-rollercan.elf /dev/ttyACM0
```

Use your actual serial port (often `/dev/cu.usbmodem…` on macOS). Flashing replaces
the NanoC6 application. See the [NanoC6 build/flash notes](../../../examples/m5nanoc6/README.md).

The protocol range is not a sustainable power rating. M5Stack specifies reduced
torque on Grove 5 V power (0.021 N·m), versus externally powered operation. Phase
current limits do not cap NanoC6 USB input current. If communication fails, the
example attempts to disable output and halts; if that write fails, the motor can
retain its last command, so disconnect its power. This is a supervised bench
example, not a hardware stop interlock.

## Protocol sources

The [product page](https://docs.m5stack.com/en/unit/Unit-RollerCAN) links an older
register sheet and a later “I2C User Manual” containing conflicting packet/baud
descriptions. This driver uses the actual application register implementation
and M5Stack's I2C library:

- [RollerCAN firmware register handler, pinned revision](https://github.com/m5stack/M5Unit-RollerCAN-Internal-FW/blob/addac0f416ceae106404aaab7da50119fae0b41a/code/Unit-RollerCAN/Core/User/i2c/i2c_protocol.c)
- [M5Stack I2C driver](https://github.com/m5stack/M5Unit-Roller/blob/main/src/unit_rolleri2c.cpp)
- [Shared protocol definitions](https://github.com/m5stack/M5Unit-Roller/blob/main/src/unit_roller_common.hpp)

Tests check wire bytes, signed and unsigned boundaries, every application
register, error propagation, address transitions, and old-firmware rejection.
The frontend corpus checks execution through Renvo, and the NanoC6 acceptance
test checks ELF compilation. These do not replace testing with a physical unit.

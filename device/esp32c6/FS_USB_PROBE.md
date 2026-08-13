# ESP32-C6 full-speed PHY probe

The hardware probe in `sandbox/esp32c6-fs-nak-sie.S` demonstrates a
software-only full-speed USB path on an M5NanoC6. It retains the internal USB
analog PHY, receives D-/D+ through the dedicated-GPIO input CSRs, and transmits
pre-encoded symbols through `USB_SERIAL_JTAG_TEST_REG` from the HP core.

Measured working properties:

- Linux accepts the custom USB 1.1 device descriptor `dead:00c6` at a newly
  assigned address.
- Linux accepts the nine-byte configuration descriptor
  `09 02 09 00 00 01 00 80 32`.
- A saturated 160 MHz store stream with a 3-cell `75/75/100 ns` pattern gives
  the required 83.333 ns average bit period.
- The PHY-test write path queues one more write than expected at EOP. Two SE0
  symbol writes after the initial SE0 store, followed by J and release, produce
  host-accepted EOPs.
- Dedicated-GPIO input taps coexist with the enabled USB analog pad function.

The probe is research code, not yet a production endpoint engine. The current
remaining issue is making all status-stage ACKs deterministic across the
undocumented APB-to-PHY phase; USB retry-aware state transitions and four ACK
cadence variants are included for continued testing.

Build the checked probe without touching hardware:

```sh
ACK_ROTATE=1 ACK_RELEASE=0 ACK_PHASE=0 FULL_PHASE=0 REL18=0 RELCONFIG=0 \
EOP_EXTRA_SE0=1 TEST_PATTERN=1 TX_CPU160=1 TX_PERIOD=3 \
TX_PATTERN_NOPS=2 STOP_AFTER_FIRST=0 RESPONSE_OFFSET=30 \
WATCHDOG_TICKS=60000 BUILD_ONLY=1 \
./sandbox/run-esp32c6-fs-romload.sh
```

The runner loads only to RAM and arms a recovery watchdog. If a timing
experiment prevents the ROM USB port from returning, a host deep suspend wakes
the board and restores the built-in USB Serial/JTAG personality.

# Tab5 Keyboard terminal

Attach the [Tab5 Keyboard](https://docs.m5stack.com/en/tab5/Tab5_Keyboard)
to Ext.Port1 (SDA GPIO0, SCL GPIO1, default I2C address `0x6d`). The example
polls the queue; GPIO50's interrupt is not needed. Select **Tab5 Keyboard
terminal** under M5Stack Tab5 in the browser examples, or compile:

```sh
go build -o sandbox/renvo ./cmd/renvo
sandbox/renvo -backend backends/esp32p4.rtg -t esp32p4/riscv32 -tags m5tab5 \
  -o sandbox/tab5-keyboard-terminal.elf ./examples/device/tab5_keyboard_terminal
```

The local-echo terminal starts landscape. **Ctrl+O** switches to portrait and
back, **Ctrl+L** clears, and touch-drag navigates scrollback. Sym/Aa translation
is handled by firmware; Ctrl and Alt modifiers, Escape, Tab, Enter, Backspace,
Delete and arrows produce terminal input. This is a terminal demonstration,
not an attached shell. Character mode reports presses only, without host key
repeat. Recent lines/colors survive orientation switches without text reflow;
text beyond a narrower column count is truncated.

Output is raw VT: `\r` returns to column zero, `\n` moves down one row.
Use `\r\n` for complete newlines, including mirrored `fmt.Printf` output;
`fmt.Fprintln` alone does not return the cursor to the left margin.

## Native rotated rendering

`board.Display.SetLandscape(bool)` switches the terminal at runtime; the next
flush refits its grid. `board.NewLandscapeSurface()` now returns RGB565 in native
scanout order, not a landscape RGBA staging image. Existing callers using its
drawing methods and `PresentLandscape` need no conversion step. Direct pixel
writers must use `surface.PixelOffset(x,y)` rather than logical row strides.

For Forms, construct a native surface and call `form.RotateSurface(surface,
graphics.Rotation90)` or `Rotation0`, then `form.Paint(surface)` and
`board.PresentPortrait(surface)` (the native presenter accepts either layout).
Rotation resizes/docks controls and invalidates the view. All four quarter-turns
are supported by `graphics.NewRotatedSurface` / `NewRotatedSurfaceBuffer`.
Width/Height, clips, transforms and dirty rectangles remain logical; Pixels and
Stride remain native. `NativeRect` converts damage for display drivers. A switch
repaints directly into the existing native buffer; there is no extra full-screen
allocation, conversion or framebuffer rotation pass. Use `board.Touch.Landscape`
to match standalone Forms touch input to the chosen portrait/landscape view.

## Driver

`renvo.dev/device/input/tab5keyboard` exposes Normal matrix press/release,
HID modifier/usage and Character events; interrupt configuration/status/clear,
queue count/clear, brightness, RGB modes and both RGB LEDs, firmware version,
and persistent I2C address changes. `Initialize(mode)` clears stale events but
does not change LEDs. `SetAddress` writes flash and waits 50ms; do not call it
in a polling loop. Driver calls return I2C errors unchanged.

The [official library](https://github.com/m5stack/M5Unit-KEYBOARD/blob/main/src/unit/unit_Tab5Keyboard.hpp)
and [firmware](https://github.com/m5stack/M5Tab5-Keyboard-Internal-FW/blob/main/code/Keyboard_APP/Core/User/i2c/user_i2c_callback.c)
clarify that character length includes the modifier (read exactly that many
bytes), and RGB LEDs occupy BGR registers `60–62` and `64–66`, skipping `63`.
The bootloader firmware-upload protocol is not part of this driver.

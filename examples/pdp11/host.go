package main

import (
	"renvo.dev/device/terminal"
	"renvo.dev/std/graphics"
	"renvo.dev/std/os"
	"renvo.dev/std/strconv"
	"renvo.dev/std/strings"
	"unsafe"
)

// Disk images are loaded as a private RAM copy. Guest writes affect this
// session only; the disk/CPU boundary is reusable on SDMMC later.
var diskImages [8][]byte
var console *terminal.Terminal
var window *graphics.Window
var pendingInput string
var outputBuffer []byte
var outputTail string
var script []string
var scriptAt int
var ticks int
var limit = 25000
var headless bool

type hostDisplay struct{}

func (d *hostDisplay) InitializeTerminal() (*graphics.Surface, bool) { return window.Surface(), true }
func (d *hostDisplay) PresentTerminal(s *graphics.Surface) bool      { return window.Present() }

func host_open(argc int32, argv **byte) int32 {
	args := []string{}
	if argc < 1 || argc > 256 {
		return 1
	}
	ptrs := (*[256]*byte)(unsafe.Pointer(argv))
	for i := 0; i < int(argc); i++ {
		p := (*[4096]byte)(unsafe.Pointer(ptrs[i]))
		n := 0
		for n < 4096 && p[n] != 0 {
			n++
		}
		args = append(args, string(p[:n]))
	}
	image := ""
	var paths [8]string
	for i := 1; i < len(args); i++ {
		a := args[i]
		if len(a) == 5 && strings.HasPrefix(a, "--rk") && a[4] >= '1' && a[4] <= '7' && i+1 < len(args) {
			i++
			paths[int(a[4]-'0')] = args[i]
		} else if a == "--headless" {
			headless = true
		} else if (a == "--script" || a == "--ticks") && i+1 < len(args) {
			i++
			if a == "--ticks" {
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 1 {
					return 1
				}
				limit = n
			} else {
				b, err := os.ReadFile(args[i])
				if err != nil {
					return 1
				}
				script = strings.Split(string(b), "\n")
			}
		} else if image == "" {
			image = a
		} else {
			print("Unexpected argument\n")
			return 1
		}
	}
	if image == "" {
		print("usage: pdp11 [--headless] [--ticks n] [--script file] rk0.dsk\n")
		return 1
	}
	var err error
	paths[0] = image
	for unit, path := range paths {
		if path == "" {
			continue
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			print(readErr.Error(), "\n")
			return 1
		}
		if len(data) < 512 || len(data) > 4872*512 || len(data)%512 != 0 {
			print("Invalid RK05 image size\n")
			return 1
		}
		// TUHS images omit trailing swap blocks. Expand the private RAM copy,
		// not the original file, to the controller's complete 203-cylinder disk.
		diskImages[unit] = make([]byte, len(data)+(4872*512-len(data)))
		copy(diskImages[unit], data)
	}
	outputBuffer = make([]byte, 0, 4096)
	if !headless {
		window = graphics.NewWindow(graphics.WindowOptions{Title: "Renvo PDP-11 / Unix V7", Width: 960, Height: 640})
		if window == nil {
			print("Unable to create terminal window\n")
			return 1
		}
		console, err = terminal.Start(&hostDisplay{}, terminal.Options{Scrollback: 256, Font: graphics.NewBuiltinFont(2), CellWidth: 12, CellHeight: 20, Baseline: 14})
		if err != nil {
			print(err.Error(), "\n")
			return 1
		}
	}
	return 0
}

func host_disk(unit, writing int32, offset uint32, buffer *byte, count int32) int32 {
	if unit < 0 || unit > 7 || count != 512 || offset%512 != 0 {
		return 1
	}
	diskImage := diskImages[int(unit)]
	if uint64(offset)+512 > uint64(len(diskImage)) {
		return 1
	}
	b := (*[512]byte)(unsafe.Pointer(buffer))
	if writing != 0 {
		copy(diskImage[int(offset):int(offset)+512], b[:])
		return 0
	}
	copy(b[:], diskImage[int(offset):int(offset)+512])
	return 0
}

func host_tx(ch int32) {
	c := byte(ch)
	outputBuffer = append(outputBuffer, c)
	if scriptAt < len(script) {
		outputTail += string([]byte{c})
		if len(outputTail) > 8192 {
			outputTail = outputTail[len(outputTail)-4096:]
		}
	}
	if len(outputBuffer) >= 4096 {
		host_flush()
	}
}
func host_flush() {
	if len(outputBuffer) == 0 {
		return
	}
	os.Stdout.Write(outputBuffer)
	if console != nil {
		console.Write(outputBuffer)
		console.Flush()
	}
	outputBuffer = outputBuffer[:0]
}

// Script lines are EXPECT<TAB>SEND; backslash-r/n are decoded in SEND.
// Waiting for output, instead of feeding a timer-driven stream, avoids dropping
// boot input while the guest resets its console or loads the kernel.
func host_tick(inputReady int32) int32 {
	ticks++
	host_flush()
	if headless && ticks >= limit {
		return -1
	}
	if scriptAt < len(script) && pendingInput == "" {
		line := script[scriptAt]
		if line == "" || strings.HasPrefix(line, "# ") {
			scriptAt++
		} else {
			parts := strings.Split(line, "\t")
			if len(parts) == 2 && strings.Contains(outputTail, parts[0]) {
				pendingInput = strings.ReplaceAll(strings.ReplaceAll(parts[1], "\\r", "\r"), "\\n", "\n")
				outputTail = ""
				scriptAt++
			}
		}
	}
	if window != nil {
		for {
			e, ok := window.Poll()
			if !ok {
				break
			}
			if e.Type == graphics.EventWindowClose {
				return -1
			}
			if e.Type == graphics.EventWindowResize || e.Type == graphics.EventWindowExpose {
				console.ResizeToSurface()
				console.Flush()
			}
			if e.Type == graphics.EventTextInput {
				for _, c := range []byte(e.Text) {
					if c == '\r' || c == '\n' || c == '\t' || c == 27 || c == 127 {
						continue
					}
					if e.Modifiers&graphics.ModifierControl != 0 {
						if c >= 'a' && c <= 'z' {
							c -= 96
						} else if c >= '@' && c <= '_' {
							c &= 31
						}
					}
					if c < 128 {
						pendingInput += string([]byte{c})
					}
				}
			}
			if e.Type == graphics.EventKeyDown {
				switch e.Key {
				case graphics.KeyEnter:
					pendingInput += "\r"
				case graphics.KeyBackspace:
					pendingInput += "\x7f"
				case graphics.KeyTab:
					pendingInput += "\t"
				case graphics.KeyEscape:
					pendingInput += "\x1b"
				}
				switch e.Key {
				case graphics.KeyUp:
					pendingInput += "\x1b[A"
				case graphics.KeyDown:
					pendingInput += "\x1b[B"
				case graphics.KeyLeft:
					pendingInput += "\x1b[D"
				case graphics.KeyRight:
					pendingInput += "\x1b[C"
				}
			}
		}
	}
	if inputReady != 0 && len(pendingInput) > 0 {
		c := pendingInput[0]
		pendingInput = pendingInput[1:]
		return int32(c) + 1
	}
	return 0
}
func host_close() {
	host_flush()
	if window != nil {
		window.Close()
	}
}

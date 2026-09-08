package main

import (
	"renvo.dev/device/board"
	"renvo.dev/device/esp32p4"
	"renvo.dev/device/fat32"
	"renvo.dev/device/gpio"
	"renvo.dev/device/i2c"
	"renvo.dev/device/input/tab5keyboard"
	"renvo.dev/device/terminal"
	"renvo.dev/std/graphics"
	"unsafe"
)

// A Go entrypoint runs imported board/package initializers before entering C.
func main() { pdp_run() }

// The SD image is a read-only seed. RK11 writes, including swap, affect only
// this full-capacity RAM disk. Reset discards the session, never the SD file.
const imagePath = "/RV260907/V7RK0.DSK"
const rkBytes = 4872 * 512

var disk []byte

// Guest memory must stay on-chip. Reserve the board's separate high-SRAM pool,
// leaving BSS, stack, DMA descriptors and the L2 cache partition untouched.
func host_memory(size int32) *byte {
	address, err := board.ReserveInternalMemory(uintptr(size), 64)
	if err != nil {
		print("Guest RAM: ", err.Error(), "\r\n")
		console.Flush()
		return nil
	}
	print("Guest RAM: 248 KiB internal SRAM at ", address, "\r\n")
	console.Flush()
	return (*byte)(unsafe.Pointer(address))
}

var console *terminal.Terminal
var keyboard *tab5keyboard.Device
var keyboardIRQ *esp32p4.Pin
var lastKeyboardPoll uint32
var output [4096]byte
var outputCount int
var inputQueue [256]byte
var inputRead, inputWrite uint32
var lastFrame, clockTime, clockRemainder uint32
var previousCR bool
var timer esp32p4.SystemTimer
var timerTicks, timerRemainder, timerMillis uint32

// Extend the low 32-bit 16 MHz counter before converting units. Dividing the
// raw counter first would wrap milliseconds every 268 seconds, not at 2^32.
// Call at least once per hardware-counter period (the CPU polls far faster).
func milliseconds() uint32 {
	ticks := timer.Ticks()
	elapsed := ticks - timerTicks
	timerTicks = ticks
	timerMillis += elapsed / 16000
	timerRemainder += elapsed % 16000
	timerMillis += timerRemainder / 16000
	timerRemainder %= 16000
	return timerMillis
}

var bootStage int
var testMode bool

func host_test() { testMode = true }

var progressTime uint32
var progressSteps uint32
var hostMillis uint32

func host_progress(pc, psw, fault int32) {
	progressSteps += 1024
	now := milliseconds()
	if now-progressTime > 10000 {
		progressTime = now
		print("\r\nTRACE steps=", progressSteps, " PC=", pc, " PSW=", psw, " fault=", fault, " queued=", inputWrite-inputRead, " host-ms=", hostMillis, "\r\n")
	}
}

func queueText(text string) {
	for i := 0; i < len(text) && inputWrite-inputRead < 256; i++ {
		inputQueue[inputWrite%256] = text[i]
		inputWrite++
	}
}

// C int is 32 bits on both hosts; use explicit widths at every C/Go boundary.
func host_open() int32 {
	if err := esp32p4.UseFullPLLClock(); err != nil {
		print("CPU clock: ", err.Error(), "\n")
		return 1
	}
	board.Display.SetLandscape(true)
	var err error
	console, err = terminal.Start(&board.Display, terminal.Options{
		Scrollback: 128, Font: graphics.NewBuiltinFont(2), CellWidth: 12, CellHeight: 20, Baseline: 14,
		FlushPolicy: terminal.FlushManual, Clock: &board.Display, Pointer: &board.Touch,
	})
	if err != nil {
		print("Terminal: ", err.Error(), "\n")
		return 1
	}
	print("Renvo PDP-11 / Unix V7\r\nCPU: full CPLL clock. Guest RAM: internal SRAM.\r\nCtrl+O: rotate. Guest writes are RAM-only; reset discards changes.\r\n")
	console.Flush()
	keyboard = tab5keyboard.New(i2c.New(board.ExtPort1()))
	keyboardIRQ = esp32p4.GPIO(50)
	if err = keyboardIRQ.Configure(gpio.Config{Direction: gpio.Input, Pull: gpio.PullUp}); err != nil {
		print("Keyboard IRQ: ", err.Error(), "\r\n")
		console.Flush()
		return 1
	}
	if err = keyboard.Initialize(tab5keyboard.Character); err != nil {
		print("Keyboard: ", err.Error(), "\r\n")
		console.Flush()
		return 1
	}
	card, err := board.OpenSD()
	if err != nil {
		print("SD: ", err.Error(), "\r\n")
		console.Flush()
		return 1
	}
	volume, err := fat32.Mount(card)
	if err != nil {
		print("FAT32: ", err.Error(), "\r\n")
		console.Flush()
		return 1
	}
	file, err := volume.Open(imagePath)
	if err != nil {
		print("Copy a V7 RK05 image to ", imagePath, "\r\n", err.Error(), "\r\n")
		console.Flush()
		return 1
	}
	size := int(file.Size())
	if size < 512 || size > rkBytes || size%512 != 0 {
		print("Invalid RK05 image size\r\n")
		console.Flush()
		return 1
	}
	disk = make([]byte, size+(rkBytes-size))
	print("Loading ", imagePath, " (", size, " bytes) via SDMMC DMA...\r\n")
	console.Flush()
	start := milliseconds()
	for offset := 0; offset < size; {
		end := offset + 32768
		if end > size {
			end = size
		}
		n, e := file.Read(disk[offset:end])
		if e != nil || n == 0 {
			print("Image read failed at ", offset, "\r\n")
			console.Flush()
			return 1
		}
		offset += n
	}
	print("Loaded in ", milliseconds()-start, " ms. Autobooting V7 (allow about a minute)...\r\n\r\n")
	if testMode {
		checksum := uint32(2166136261)
		for i := 0; i < size; i++ {
			checksum = (checksum ^ uint32(disk[i])) * 16777619
		}
		print("SD image FNV-1a32: ", checksum, "\r\n")
	}
	console.Flush()
	lastFrame = milliseconds()
	clockTime = lastFrame
	return 0
}

// The core uses complete 512-byte host transactions, even for partial DMA.
// Bounds are checked before forming the C buffer view or indexing the RAM disk.
func host_disk(unit, writing int32, offset uint32, buffer *byte, count int32) int32 {
	if unit != 0 || count != 512 || len(disk) < 512 || offset%512 != 0 || offset > uint32(len(disk))-512 {
		return -1
	}
	b := (*[512]byte)(unsafe.Pointer(buffer))
	for i := 0; i < 512; i++ {
		if writing != 0 {
			disk[int(offset)+i] = b[i]
		} else {
			b[i] = disk[int(offset)+i]
		}
	}
	return 0
}

func flushOutput() {
	if outputCount != 0 {
		// print mirrors once into the terminal and also sends a serial transcript.
		// Preserve guest CR/LF exactly; the guest TTY owns output line discipline.
		print(string(output[:outputCount]))
		outputCount = 0
	}
}
func host_tx(ch int32) {
	// Feed the actual on-disk bootstrap, not a preloaded kernel. Stop automating
	// after the two V7 loader prompts so all subsequent input belongs to the user.
	if bootStage == 0 && ch == 64 {
		queueText("boot\r")
		bootStage = 1
	} else if bootStage == 1 && ch == 58 {
		queueText("rk(0,0)rkunix\r")
		bootStage = 2
	} else if bootStage == 2 && ch == 35 {
		bootStage = 3
		if testMode {
			queueText("echo RENVO_P4_V7_OK; ls /; echo RENVO_RAM_WRITE_OK > /renvo; cat /renvo; rm /renvo\r")
		}
	}
	output[outputCount] = byte(ch)
	outputCount++
	if outputCount == len(output) {
		flushOutput()
	}
}

// Accumulate fractional milliseconds at 60 Hz; never replay a long pause as
// an interrupt storm. Unsigned elapsed subtraction also tolerates timer wrap.
func host_clock() int32 {
	now := milliseconds()
	elapsed := now - clockTime
	clockTime = now
	if elapsed > 100 {
		elapsed = 100
	}
	clockRemainder += elapsed * 60
	if clockRemainder >= 1000 {
		clockRemainder %= 1000
		return 1
	}
	return 0
}

func host_tick(ready int32) int32 {
	now := milliseconds()
	if now-lastFrame >= 16 {
		lastFrame = now
		flushOutput()
		var bytes [16]byte
		// Reserve room for one entire keyboard event; do not drop an escape tail.
		// Use the keyboard's IRQ rather than bit-banging an empty I2C read
		// every frame. A slow fallback poll covers missed/edge-only events.
		if !keyboardIRQ.Get() || now-lastKeyboardPoll >= 250 {
			lastKeyboardPoll = now
			for batch := 0; batch < 32 && inputWrite-inputRead <= 240; batch++ {
				event, ok, err := keyboard.NextCharacter()
				if err != nil || !ok {
					break
				}
				n := event.TerminalBytes(bytes[:])
				if n == 1 && bytes[0] == 15 {
					board.Display.SetLandscape(!board.Display.Landscape)
				} else {
					for i := 0; i < n; i++ {
						ch := bytes[i]
						// Character-mode Enter is CRLF. Deliver one CR to the Unix TTY.
						if ch == 10 && previousCR {
							previousCR = false
							continue
						}
						previousCR = ch == 13
						if ch == 10 {
							ch = 13
						}
						if ch == 8 {
							ch = 127
						}
						inputQueue[inputWrite%256] = ch
						inputWrite++
					}
				}
			}
		}
		if !console.Poll() || !console.Flush() {
			return -1
		}
	}
	if ready != 0 && inputRead != inputWrite {
		ch := inputQueue[inputRead%256]
		inputRead++
		if testMode {
			hostMillis += milliseconds() - now
		}
		return int32(ch) + 1
	}
	if testMode {
		hostMillis += milliseconds() - now
	}
	return 0
}

func host_stop(pc, psw, fault int32) {
	flushOutput()
	print("\r\nPDP stopped: PC=", pc, " PSW=", psw, " fault=", fault, "\r\n")
	console.Flush()
}

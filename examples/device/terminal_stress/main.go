package main

import (
	"renvo.dev/device/board"
	"renvo.dev/device/i2c"
	"renvo.dev/device/sensor/adxl345"
	"renvo.dev/device/terminal"
	"renvo.dev/std/fmt"
)

func nextRandom(state *uint32) uint32 {
	value := *state
	value ^= value << 13
	value ^= value >> 17
	value ^= value << 5
	*state = value
	return value
}

func drainKeyboard(console *terminal.Terminal) {
	var input [16]byte
	for {
		count, _ := console.Read(input[:])
		for index := 0; index < count; index++ {
			fmt.Fprintf(console, "\x1b[1;7m KEY byte=%d \x1b[0m\r\n", input[index])
		}
		if count < len(input) {
			return
		}
	}
}

func setStressStyle(console *terminal.Terminal, sequence, random uint32) {
	switch sequence & 3 {
	case 0:
		fmt.Fprintf(console, "\x1b[1;38;5;%dm", 16+random%216)
	case 1:
		fmt.Fprintf(console, "\x1b[4;38;2;%d;%d;%dm", random&255, random>>8&255, random>>16&255)
	case 2:
		fmt.Fprintf(console, "\x1b[7;38;5;%d;48;5;%dm", 16+random%216, 16+(random>>8)%216)
	default:
		console.WriteString("\x1b[1;4;33m")
	}
}

func main() {
	if err := board.StartTerminal(); err != nil {
		for {
			print("Tab5 terminal stress initialization failed: ", err.Error(), "\n")
		}
	}
	console := board.Console

	bus := i2c.New(board.PortA())
	sensor := adxl345.New(bus, adxl345.AddressLow)
	sensorErr := sensor.Initialize()
	if sensorErr != nil {
		sensor = adxl345.New(bus, adxl345.AddressHigh)
		sensorErr = sensor.Initialize()
	}
	console.Reset()
	fmt.Fprintf(console, "\x1b[1;31mTERMINAL STRESS\x1b[0m: variable bursts, wrapping, ANSI, DMA scroll, touch and I2C\r\n")
	if sensorErr != nil {
		fmt.Fprintf(console, "\x1b[31mADXL345 unavailable: %s\x1b[0m\r\n", sensorErr)
	}
	console.Flush()

	randomState := uint32(0x6d2b79f5)
	frame := uint32(0)
	sequence := uint32(0)
	fps := uint32(0)
	windowFrames := uint32(0)
	windowStarted := board.Milliseconds()
	var reading adxl345.Reading

	for board.TickTerminal() {
		drainKeyboard(console)
		if sensorErr == nil {
			sensorErr = sensor.ReadInto(&reading)
		}
		frame++
		windowFrames++
		now := board.Milliseconds()
		elapsed := now - windowStarted
		if elapsed >= 1000 {
			fps = windowFrames * 1000 / elapsed
			windowFrames = 0
			windowStarted = now
		}

		random := nextRandom(&randomState)
		burst := 2 + int(random&3)
		for line := 0; line < burst; line++ {
			random = nextRandom(&randomState)
			setStressStyle(console, sequence, random)
			stats := console.Statistics()
			display := board.FramebufferStats()
			fmt.Fprintf(console,
				"F%06d L%07d fps=%d last=%dms burst=%d dma=%d under=%d X=%d Y=%d Z=%d :: abcdefghijklmnopqrstuvwxyz0123456789",
				frame, sequence, fps, stats.LastFrameMillis, burst,
				display.DMA2DCopies, display.ScanoutUnderruns,
				reading.X, reading.Y, reading.Z,
			)
			if sequence&7 == 0 {
				console.WriteString("\tTAB\b!")
			}
			console.WriteString("\x1b[0m\x1b[K\r\n")
			sequence++
		}
		if frame%60 == 0 {
			// This deliberately exercises the stdout mirror and USB serial path in
			// addition to direct terminal writes, but only once per second.
			fmt.Printf("stress heartbeat frame=%d lines=%d fps=%d\n", frame, sequence, fps)
		}
		console.Flush()
	}
}

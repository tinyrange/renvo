package rp2

import (
	"renvo.dev/device/gpio"
	"renvo.dev/device/mmio"
	"renvo.dev/device/ws2812"
)

const (
	pio0Base       = uintptr(0x50200000)
	pioControl     = pio0Base + 0x000
	pioFDebug      = pio0Base + 0x008
	pioTXFIFO0     = pio0Base + 0x010
	pioInstruction = pio0Base + 0x048
	pioSM0ClockDiv = pio0Base + 0x0c8
	pioSM0Exec     = pio0Base + 0x0cc
	pioSM0Shift    = pio0Base + 0x0d0
	pioSM0Instr    = pio0Base + 0x0d8
	pioSM0Pin      = pio0Base + 0x0dc

	pioFunction        = uint32(6)
	pioSM0Enable       = uint32(1)
	pioSM0Restart      = uint32(1 << 4)
	pioSM0ClockRestart = uint32(1 << 8)
	pioTXFull          = uint32(1 << 16)
	pioTXStall         = uint32(1 << 24)
	pioAutoPull        = uint32(1 << 17)
	pioJoinTX          = uint32(1 << 30)
	pioJoinRX          = uint32(1 << 31)
	pioPullThreshold24 = uint32(24 << 25)
	pioSideSetCount1   = uint32(1 << 29)
	pioSetCount1       = uint32(1 << 26)
	// RP2 programs currently retain the ROM's 12 MHz system clock. The PIO
	// program consumes ten 8 MHz cycles per bit, producing the WS2812 800 kHz
	// waveform with the standard T1/T2/T3 timing split.
	pioClockDiv8MHz = uint32(1<<16 | 128<<8)
	ws2812PollLimit = 65536
)

func releasePIO0Reset() {
	releaseReset(rp2040PIO0Reset, rp2350PIO0Reset)
}

var ws2812Program = [4]uint32{
	0x6221, // out x, 1       side 0 [2]
	0x1123, // jmp !x, 3      side 1 [1]
	0x1400, // jmp 0          side 1 [4]
	0xa442, // nop            side 0 [4]
}

type ws2812PIO struct {
	data        *Pin
	power       gpio.Pin
	initialized bool
}

// WS2812Transmitter selects PIO state machine zero for pixels attached to this
// pin. Portable applications reach it through ws2812.Output.
func (p *Pin) WS2812Transmitter(power gpio.Pin) ws2812.Transmitter {
	return &ws2812PIO{data: p, power: power}
}

func (p *ws2812PIO) initialize() {
	if p.initialized {
		return
	}
	if p.power != nil {
		p.power.Set(true)
		_ = p.power.Configure(gpio.Config{Direction: gpio.Output})
	}
	releasePIO0Reset()

	// Stop SM0 while replacing its program and pin routing. PIO0/SM0 is the
	// board-level WS2812 resource; the portable driver owns it for the lifetime
	// of the strip.
	mmio.Store32(pioControl, mmio.Load32(pioControl)&^pioSM0Enable)
	for index := 0; index < len(ws2812Program); index++ {
		mmio.Store32(pioInstruction+uintptr(index*4), ws2812Program[index])
	}
	mmio.Store32(pioSM0ClockDiv, pioClockDiv8MHz)
	mmio.Store32(pioSM0Exec, 3<<12) // wrap bottom 0, wrap top 3
	// Match the SDK WS2812 configuration. Joining both FIFO halves for TX also
	// makes toggling the opposite join bit a defined way to discard stale words
	// left by an application interrupted during a hot reload.
	mmio.Store32(pioSM0Shift, pioAutoPull|pioPullThreshold24|pioJoinTX)
	mmio.Store32(pioSM0Shift, mmio.Load32(pioSM0Shift)^pioJoinRX)
	mmio.Store32(pioSM0Shift, mmio.Load32(pioSM0Shift)^pioJoinRX)
	mmio.Store32(pioSM0Pin,
		uint32(p.data.number)<<5|uint32(p.data.number)<<10|pioSetCount1|pioSideSetCount1)

	// FUNCSEL 6 is PIO0 on both RP2040 and RP2350. The immediate SET executes
	// while stopped so the state machine, rather than SIO, owns output enable.
	mmio.Store32(p.data.control(), pioFunction)
	mmio.Store32(pioSM0Instr, 0xe081) // set pindirs, 1
	mmio.Store32(pioSM0Instr, 0xe000) // set pins, 0
	mmio.Store32(pioSM0Instr, 0x0000) // jmp 0
	mmio.Store32(pioFDebug, pioTXStall)
	mmio.Store32(pioControl, mmio.Load32(pioControl)|pioSM0Restart|pioSM0ClockRestart|pioSM0Enable)
	p.initialized = true
}

func (p *ws2812PIO) Transmit(data []byte) bool {
	if len(data)%3 != 0 {
		return false
	}
	p.initialize()
	for index := 0; index < len(data); index += 3 {
		ready := false
		for attempt := 0; attempt < ws2812PollLimit; attempt++ {
			if mmio.Load32(pio0Base+0x004)&pioTXFull == 0 {
				ready = true
				break
			}
		}
		if !ready {
			return false
		}
		word := uint32(data[index])<<24 | uint32(data[index+1])<<16 | uint32(data[index+2])<<8
		mmio.Store32(pioTXFIFO0, word)
	}
	// Clear the old empty-FIFO indication only after publishing the final word.
	// Clearing it first races the already-stalled OUT instruction, which can
	// immediately reassert TXSTALL before the FIFO write and make a caller treat
	// an incomplete frame as finished.
	mmio.Store32(pioFDebug, pioTXStall)
	finished := false
	for attempt := 0; attempt < ws2812PollLimit; attempt++ {
		if mmio.Load32(pioFDebug)&pioTXStall != 0 {
			finished = true
			break
		}
	}
	if !finished {
		return false
	}
	Clock{}.DelayMicroseconds(80)
	return true
}

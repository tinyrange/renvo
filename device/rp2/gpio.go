// Package rp2 provides peripherals shared by RP2040 and RP2350 Arm images.
package rp2

import (
	"renvo.dev/device/gpio"
	"renvo.dev/device/mmio"
)

const (
	cpuid           = uintptr(0xe000ed00)
	sioBase         = uintptr(0xd0000000)
	gpioInput       = sioBase + 0x04
	gpioOutSet      = sioBase + 0x18
	gpioOutClear    = sioBase + 0x20
	gpioEnableSet   = sioBase + 0x38
	gpioEnableClear = sioBase + 0x40
	padOutputOff    = uint32(1 << 7)
	padInput        = uint32(1 << 6)
	padPullUp       = uint32(1 << 3)
	padPullDown     = uint32(1 << 2)
	gpioFunction    = uint32(5)
)

var gpioResetReleased bool

func releaseGPIOReset() {
	if gpioResetReleased {
		return
	}
	releaseReset(rp2040IOBank0Reset|rp2040PadsBank0Reset,
		rp2350IOBank0Reset|rp2350PadsBank0Reset)
	gpioResetReleased = true
}

func isRP2350() bool { return (mmio.Load32(cpuid)>>4)&0xfff == 0xd21 }

func ioBank0Base() uintptr {
	if isRP2350() {
		return 0x40028000
	}
	return 0x40014000
}

func padsBank0Base() uintptr {
	if isRP2350() {
		return 0x40038000
	}
	return 0x4001c000
}

// Pin identifies a GPIO present on either RP2 chip family.
type Pin struct {
	number      uint8
	initialized bool
}

// GPIO returns the concrete pin capability for number.
func GPIO(number uint8) *Pin { return &Pin{number: number} }

func (p *Pin) bit() uint32 { return uint32(1) << uint(p.number) }

func (p *Pin) control() uintptr { return ioBank0Base() + uintptr(p.number)*8 + 4 }
func (p *Pin) pad() uintptr     { return padsBank0Base() + uintptr(p.number)*4 + 4 }

// Configure selects the SIO function and applies direction and pull policy.
func (p *Pin) Configure(config gpio.Config) error {
	releaseGPIOReset()
	ioBase := uintptr(0x40014000)
	padsBase := uintptr(0x4001c000)
	if isRP2350() {
		ioBase = 0x40028000
		padsBase = 0x40038000
	}
	padAddress := padsBase + uintptr(p.number)*4 + 4
	controlAddress := ioBase + uintptr(p.number)*8 + 4
	mask := padOutputOff | padInput | padPullUp | padPullDown
	if isRP2350() {
		mask |= 1 << 8
	}
	pad := mmio.Load32(padAddress) &^ mask
	pad |= padInput
	if config.Pull == gpio.PullUp {
		pad |= padPullUp
	} else if config.Pull == gpio.PullDown {
		pad |= padPullDown
	}
	mmio.Store32(padAddress, pad)
	mmio.Store32(controlAddress, gpioFunction)
	if config.Direction == gpio.Output {
		mmio.Store32(gpioEnableSet, p.bit())
	} else {
		mmio.Store32(gpioEnableClear, p.bit())
	}
	p.initialized = true
	return nil
}

// Set changes the output latch without changing direction.
func (p *Pin) Set(high bool) {
	if !p.initialized {
		_ = p.Configure(gpio.Config{Direction: gpio.Output})
	}
	if high {
		mmio.Store32(gpioOutSet, p.bit())
	} else {
		mmio.Store32(gpioOutClear, p.bit())
	}
}

// Toggle changes the pin's output latch.
func (p *Pin) Toggle() {
	p.Set(!p.Get())
}

// Get samples the GPIO input register.
func (p *Pin) Get() bool { return mmio.Load32(gpioInput)&p.bit() != 0 }

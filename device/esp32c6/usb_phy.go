package esp32c6

import (
	"renvo.dev/device/mmio"
	"renvo.dev/device/usb/lowspeed"
)

const (
	usbSerialJTAGConf0 = uintptr(0x6000f018)
	usbSerialJTAGTest  = uintptr(0x6000f01c)
	usbDPIOMux         = uintptr(0x6009003c) // GPIO13
	usbDMIOMux         = uintptr(0x60090038) // GPIO12
	cpuGPIOInput0      = uintptr(0x600911c4)
	cpuGPIOInput1      = uintptr(0x600911c8)
	rmtInputSelect0    = uintptr(0x60091270) // RMT_SIG_IN0_IDX
	rmtRXConfig0       = uintptr(0x60006018)
	rmtRXConfig1       = uintptr(0x6000601c)
	rmtRXStatus        = uintptr(0x60006030)
	rmtRXMemory        = uintptr(0x60006580)
	rmtTXConfig0       = uintptr(0x60006010)
	rmtTXConfig1       = uintptr(0x60006014)
	rmtTXSim           = uintptr(0x6000606c)
	rmtTXMemory0       = uintptr(0x60006400)
	rmtTXMemory1       = uintptr(0x600064c0)

	usbPadPullOverride = uint32(1 << 8)
	usbDPPullUp        = uint32(1 << 9)
	usbDPPullDown      = uint32(1 << 10)
	usbDMPullUp        = uint32(1 << 11)
	usbDMPullDown      = uint32(1 << 12)
	usbPadEnable       = uint32(1 << 14)

	usbTestEnable = uint32(1)
	usbTestOutput = uint32(1 << 1)
	usbTestTXDP   = uint32(1 << 2)
	usbTestTXDM   = uint32(1 << 3)
	usbTestRXDP   = uint32(1 << 5)
	usbTestRXDM   = uint32(1 << 6)

	rmtRXDone           = uint32(1 << 2)
	rmtRXMemoryBase     = uint32(2 * 48)
	rmtRXMemoryWords    = uint32(48)
	rmtRXIdleTicks      = uint32(400)
	rmtRXInputEnable    = uint32(1 << 7)
	rmtRXPin            = uint32(13)
	rmtRXMemoryOwner    = uint32(1 << 3)
	rmtRXConfigUpdate   = uint32(1 << 15)
	rmtRXEnable         = uint32(1)
	rmtRXResetWrite     = uint32(1 << 1)
	rmtRXResetAPB       = uint32(1 << 2)
	rmtRXMemoryBlock    = uint32(1 << 23)
	rmtRXCaptureTimeout = uint32(8000000) // 500 ms at 16 MHz
	rmtTXDoneBoth       = uint32(3)
	rmtTXDivider        = uint32(1 << 8)
	rmtTXIdleEnable     = uint32(1 << 6)
	rmtTXIdleHigh       = uint32(1 << 5)
	rmtTXMemoryBlock    = uint32(1 << 16)
	rmtTXUpdate         = uint32(1 << 24)
	rmtTXSimultaneous   = uint32(7)
	rmtTXTimeout        = uint32(16000) // one millisecond at 16 MHz

	// USB hosts require at least 2.5 microseconds of single-ended zero before
	// treating a pull-up change as a detach followed by a new attachment. A
	// longer interval also gives the fixed-function controller time to stop
	// driving before software selects the opposite-speed pull-up.
	usbDetachMilliseconds = uint32(10)
)

// USBPHY exposes the ESP32-C6 USB Serial/JTAG block's documented raw full-
// speed PHY test interface. The software protocol may operate at low speed;
// no programmable USB controller is assumed.
type USBPHY struct {
	originalConf  uint32
	originalDPMux uint32
	originalDMMux uint32
	originalCPU0  uint32
	originalCPU1  uint32
	bitDelay      uint32
	hardwareTX    bool
	owned         bool
	dedicatedRX   bool
	resetSeen     bool
	receivedPID   byte
	receivedData  [16]byte
	receivedLen   int
	receivedOK    bool
}

func usbLine(dp, dm bool) (byte, bool) {
	if !dp && !dm {
		return 0, true
	}
	if !dp && dm {
		return 1, true
	}
	if dp && !dm {
		return 2, true
	}
	return 0, false
}

func decodeRMTPacket(states, data []byte, words []uint32) (count int, pid byte, length int) {
	if len(states) < 11 || len(words) == 0 {
		return 0, 0, 0
	}
	count = 0
	for wordIndex := 0; wordIndex < len(words); wordIndex++ {
		word := words[wordIndex]
		for half := 0; half < 2; half++ {
			duration := word & 0x7fff
			line := byte(1)
			if word&(1<<15) != 0 {
				line = 2
			}
			word >>= 16
			if duration == 0 {
				continue
			}
			// 80 MHz / 1.5 MHz is 53 1/3 RMT ticks per bit cell.
			cells := int((duration + 26) / 53)
			for repeat := 0; repeat < cells && count+3 <= len(states); repeat++ {
				states[count] = line
				count++
				if count < 8 || count+3 > len(states) {
					continue
				}
				states[count], states[count+1], states[count+2] = 0, 0, 1
				decodedPID, decodedLength, ok := lowspeed.Decode(data, states[:count+3])
				if ok {
					return count + 3, decodedPID, decodedLength
				}
			}
		}
	}
	return 0, 0, 0
}

// NewUSBPHY returns an instruction-timed raw USB PHY for emulator and
// isolated-fixture qualification.
func NewUSBPHY(delayIterations uint32) USBPHY {
	return USBPHY{bitDelay: delayIterations}
}

// NewRMTUSBPHY returns a raw USB PHY with hardware-timed transmission.
func NewRMTUSBPHY() USBPHY { return USBPHY{hardwareTX: true} }

// Takeover immediately selects the raw PHY test path and internal low-speed D-
// pull-up. It is intended for isolated electrical fixtures and emulators; use
// TakeoverDetached when changing personalities on a live USB bus.
func (p *USBPHY) Takeover() {
	if p.owned {
		return
	}
	p.originalConf = mmio.Load32(usbSerialJTAGConf0)
	config := p.originalConf &^ (usbDPPullUp | usbDPPullDown | usbDMPullUp | usbDMPullDown)
	config |= usbPadPullOverride | usbDMPullUp | usbPadEnable
	mmio.Store32(usbSerialJTAGConf0, config)
	mmio.Store32(usbSerialJTAGTest, usbTestEnable)
	p.resetSeen = false
	p.owned = true
}

// TakeoverDetached disconnects the fixed full-speed Serial/JTAG personality
// before attaching the raw low-speed PHY on a live USB bus.
func (p *USBPHY) TakeoverDetached() {
	if p.owned {
		return
	}
	p.originalConf = mmio.Load32(usbSerialJTAGConf0)
	config := p.originalConf &^ (usbDPPullUp | usbDPPullDown | usbDMPullUp | usbDMPullDown)
	config |= usbPadPullOverride | usbPadEnable
	mmio.Store32(usbSerialJTAGConf0, config)
	mmio.Store32(usbSerialJTAGTest, usbTestEnable)
	timer := SystemTimer{}
	timer.DelayMilliseconds(usbDetachMilliseconds)
	mmio.Store32(usbSerialJTAGConf0, config|usbDMPullUp)
	p.resetSeen = false
	p.owned = true
}

// TakeoverFullSpeedDetached disconnects USB Serial/JTAG and reattaches the raw
// PHY with the internal full-speed D+ pull-up. PHY-test mode retains the C6's
// analog receiver, transmitter, and USB pad electrical characteristics.
func (p *USBPHY) TakeoverFullSpeedDetached() {
	if p.owned {
		return
	}
	p.originalConf = mmio.Load32(usbSerialJTAGConf0)
	config := p.originalConf &^ (usbDPPullUp | usbDPPullDown | usbDMPullUp | usbDMPullDown)
	config |= usbPadPullOverride | usbPadEnable
	mmio.Store32(usbSerialJTAGConf0, config)
	mmio.Store32(usbSerialJTAGTest, usbTestEnable)
	timer := SystemTimer{}
	timer.DelayMilliseconds(usbDetachMilliseconds)
	mmio.Store32(usbSerialJTAGConf0, config|usbDPPullUp)
	p.resetSeen = false
	p.owned = true
}

// EnableDedicatedUSBInputs taps D-/D+ into bits 0/1 of the CPU dedicated-GPIO
// input CSR without changing either pad's USB IOMUX function. Release restores
// every modified register.
func (p *USBPHY) EnableDedicatedUSBInputs() bool {
	if !p.owned || p.dedicatedRX {
		return p.owned
	}
	p.originalDMMux = mmio.Load32(usbDMIOMux)
	p.originalDPMux = mmio.Load32(usbDPIOMux)
	p.originalCPU0 = mmio.Load32(cpuGPIOInput0)
	p.originalCPU1 = mmio.Load32(cpuGPIOInput1)
	mmio.Store32(usbDMIOMux, p.originalDMMux|gpioInputEnable)
	mmio.Store32(usbDPIOMux, p.originalDPMux|gpioInputEnable)
	mmio.Store32(cpuGPIOInput0, 12|1<<7)
	mmio.Store32(cpuGPIOInput1, 13|1<<7)
	p.dedicatedRX = true
	return true
}

// Release restores the fixed USB Serial/JTAG function for programming and
// debugging. It is safe to call more than once.
func (p *USBPHY) Release() {
	if !p.owned {
		return
	}
	if p.dedicatedRX {
		mmio.Store32(cpuGPIOInput0, p.originalCPU0)
		mmio.Store32(cpuGPIOInput1, p.originalCPU1)
		mmio.Store32(usbDMIOMux, p.originalDMMux)
		mmio.Store32(usbDPIOMux, p.originalDPMux)
		p.dedicatedRX = false
	}
	config := mmio.Load32(usbSerialJTAGConf0)
	config &^= usbDPPullUp | usbDMPullUp
	mmio.Store32(usbSerialJTAGConf0, config)
	mmio.Store32(usbSerialJTAGTest, 0)
	timer := SystemTimer{}
	timer.DelayMilliseconds(usbDetachMilliseconds)
	mmio.Store32(usbSerialJTAGConf0, p.originalConf)
	p.resetSeen = false
	p.owned = false
}

// Sample returns the single-ended D+ and D- receiver states.
func (p *USBPHY) Sample() (dp, dm bool) {
	value := mmio.Load32(usbSerialJTAGTest)
	return value&usbTestRXDP != 0, value&usbTestRXDM != 0
}

// Receive waits for one low-speed packet or a bus reset. States receives one
// sampled line state per bit cell, including SYNC and EOP. A true reset result
// has no packet data. RMT captures edge timing while the system timer separates
// a two-cell EOP from a sustained host reset.
func (p *USBPHY) Receive(states []byte) (count int, reset bool) {
	p.receivedOK = false
	p.receivedLen = 0
	if !p.owned || len(states) == 0 {
		return 0, false
	}
	timer := SystemTimer{}
	if !p.resetSeen {
		for {
			dp, dm := p.Sample()
			line, valid := usbLine(dp, dm)
			if !valid || line != 0 {
				continue
			}
			started := timer.Ticks()
			for line == 0 {
				dp, dm = p.Sample()
				line, valid = usbLine(dp, dm)
				if !valid {
					line = 0
				}
				// EOP is only two cells; a host reset holds SE0 for at least
				// 10 ms. One millisecond is therefore a conservative, cheap
				// discriminator even with peripheral timer-read latency.
				if timer.Ticks()-started >= 16000 {
					for line == 0 {
						dp, dm = p.Sample()
						line, valid = usbLine(dp, dm)
						if !valid {
							line = 0
						}
					}
					p.resetSeen = true
					return 0, true
				}
			}
		}
	}
	// GPIO13 remains under the dedicated USB pad function, but its input also
	// feeds RMT RX channel 2. Arm immediately after reset so SYNC is not missed.
	mmio.Store32(usbDPIOMux, mmio.Load32(usbDPIOMux)|1<<9)
	mmio.Store32(rmtInputSelect0, rmtRXPin|rmtRXInputEnable)
	clock := mmio.Load32(pcrRMTConfig) | rmtClockEnable
	mmio.Store32(pcrRMTConfig, clock|rmtReset)
	mmio.Store32(pcrRMTConfig, clock&^rmtReset)
	mmio.Store32(pcrRMTClockConfig, rmtClock80MHz|rmtSourceEnable)
	mmio.Store32(rmtSystemConfig, rmtDirectMemory)
	mmio.Store32(rmtInterruptClear, uint32(0xffffffff))
	mmio.Store32(rmtRXConfig0, 1|rmtRXIdleTicks<<8|rmtRXMemoryBlock)
	mmio.Store32(rmtRXConfig1, rmtRXResetWrite|rmtRXResetAPB)
	mmio.Store32(rmtRXConfig1, rmtRXMemoryOwner)
	mmio.Store32(rmtRXConfig1, rmtRXMemoryOwner|rmtRXEnable|rmtRXConfigUpdate)
	started := timer.Ticks()
	for mmio.Load32(rmtInterruptRaw)&rmtRXDone == 0 && timer.Ticks()-started < rmtRXCaptureTimeout {
	}
	status := mmio.Load32(rmtRXStatus) & 0x1ff
	mmio.Store32(rmtRXConfig1, rmtRXConfigUpdate)
	wordCount := uint32(0)
	if status > rmtRXMemoryBase {
		wordCount = status - rmtRXMemoryBase
	}
	if wordCount > rmtRXMemoryWords {
		wordCount = rmtRXMemoryWords
	}
	var words [48]uint32
	for index := uint32(0); index < wordCount; index++ {
		words[index] = mmio.Load32(rmtRXMemory + uintptr(index)*4)
	}
	mmio.Store32(rmtInterruptClear, rmtRXDone)
	var pid byte
	var length int
	count, pid, length = decodeRMTPacket(states, p.receivedData[:], words[:wordCount])
	if count != 0 {
		p.receivedPID = pid
		p.receivedLen = length
		p.receivedOK = true
	}
	return count, false
}

// Packet returns the packet validated by the most recent successful Receive.
func (p *USBPHY) Packet(data []byte) (pid byte, length int, ok bool) {
	if !p.receivedOK || len(data) < p.receivedLen {
		return 0, 0, false
	}
	copy(data, p.receivedData[:p.receivedLen])
	return p.receivedPID, p.receivedLen, true
}

func encodeRMTSignal(words []uint32, states []byte, dp bool) int {
	var entries [96]uint32
	entryCount := 0
	for index, line := range states {
		high := line == 3
		if dp {
			high = high || line == 2
		} else {
			high = high || line == 1
		}
		duration := uint32(53)
		if index%3 == 2 {
			duration = 54
		}
		if entryCount != 0 && (entries[entryCount-1]>>15 != 0) == high {
			entries[entryCount-1] += duration
			continue
		}
		if entryCount >= len(entries)-1 {
			return 0
		}
		entries[entryCount] = duration
		if high {
			entries[entryCount] |= 1 << 15
		}
		entryCount++
	}
	// A zero-duration item terminates transmission before the channel's idle
	// level takes over.
	entryCount++
	wordCount := (entryCount + 1) / 2
	if wordCount > len(words) {
		return 0
	}
	for index := 0; index < wordCount; index++ {
		words[index] = entries[index*2]
		if index*2+1 < entryCount {
			words[index] |= entries[index*2+1] << 16
		}
	}
	return wordCount
}

// Transmit drives pre-encoded low-speed line states using two synchronized RMT
// channels. Hardware timing avoids compiler- and CPU-clock-dependent bit loops.
func (p *USBPHY) Transmit(states []byte) {
	if !p.owned {
		return
	}
	if len(states) == 0 || len(states) > 128 {
		return
	}
	if !p.hardwareTX {
		var values [128]uint32
		for index, line := range states {
			value := usbTestEnable | usbTestOutput
			value |= uint32(line&1) << 3
			value |= uint32((line>>1)&1) << 2
			values[index] = value
		}
		for index := 0; index < len(states); index++ {
			mmio.Store32(usbSerialJTAGTest, values[index])
			for delay := uint32(0); delay < p.bitDelay; delay++ {
			}
		}
		mmio.Store32(usbSerialJTAGTest, usbTestEnable)
		return
	}
	var dpWords, dmWords [48]uint32
	dpCount := encodeRMTSignal(dpWords[:], states, true)
	dmCount := encodeRMTSignal(dmWords[:], states, false)
	if dpCount == 0 || dmCount == 0 {
		return
	}

	clock := mmio.Load32(pcrRMTConfig) | rmtClockEnable
	mmio.Store32(pcrRMTConfig, clock)
	mmio.Store32(pcrRMTClockConfig, rmtClock80MHz|rmtSourceEnable)
	mmio.Store32(rmtSystemConfig, rmtDirectMemory)
	for index := 0; index < dpCount; index++ {
		mmio.Store32(rmtTXMemory0+uintptr(index)*4, dpWords[index])
	}
	for index := 0; index < dmCount; index++ {
		mmio.Store32(rmtTXMemory1+uintptr(index)*4, dmWords[index])
	}

	dpMux := mmio.Load32(usbDPIOMux)
	dmMuxAddress := usbDPIOMux - 4
	dmMux := mmio.Load32(dmMuxAddress)
	dpSelect := outputSelectBase + 13*4
	dmSelect := outputSelectBase + 12*4
	dpOutput := mmio.Load32(dpSelect)
	dmOutput := mmio.Load32(dmSelect)
	_ = GPIO(13).ConfigureOutputSignal(rmtOutputSignal)
	_ = GPIO(12).ConfigureOutputSignal(rmtOutputSignal + 1)
	mmio.Store32(usbDPIOMux, mmio.Load32(usbDPIOMux)|gpioInputEnable)

	dpConfig := rmtTXDivider | rmtTXMemoryBlock | rmtTXIdleEnable
	dmConfig := dpConfig | rmtTXIdleHigh
	mmio.Store32(rmtInterruptClear, rmtTXDoneBoth)
	mmio.Store32(rmtTXConfig0, dpConfig|rmtResetRead|rmtResetMemory)
	mmio.Store32(rmtTXConfig1, dmConfig|rmtResetRead|rmtResetMemory)
	mmio.Store32(rmtTXConfig0, dpConfig|rmtTXUpdate)
	mmio.Store32(rmtTXConfig1, dmConfig|rmtTXUpdate)
	mmio.Store32(rmtTXSim, rmtTXSimultaneous)
	mmio.Store32(rmtTXConfig0, dpConfig|rmtStart)
	mmio.Store32(rmtTXConfig1, dmConfig|rmtStart)
	timer := SystemTimer{}
	started := timer.Ticks()
	for mmio.Load32(rmtInterruptRaw)&rmtTXDoneBoth != rmtTXDoneBoth && timer.Ticks()-started < rmtTXTimeout {
	}
	mmio.Store32(rmtInterruptClear, rmtTXDoneBoth)
	mmio.Store32(rmtTXSim, 0)

	mmio.Store32(gpioEnableClear, 1<<13|1<<12)
	mmio.Store32(dpSelect, dpOutput)
	mmio.Store32(dmSelect, dmOutput)
	mmio.Store32(usbDPIOMux, dpMux)
	mmio.Store32(dmMuxAddress, dmMux)
}

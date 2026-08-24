package esp32c6

import (
	"renvo.dev/device/mmio"
	"renvo.dev/device/uart"
)

const (
	uartOneBase         = uintptr(0x60001000)
	uartOnePCRConfig    = uintptr(0x6009600c)
	uartOnePCRClock     = uintptr(0x60096010)
	uartOneOutputSignal = uint32(9)

	uartFIFO       = uintptr(0x00)
	uartClockDiv   = uintptr(0x14)
	uartStatus     = uintptr(0x1c)
	uartConfigZero = uintptr(0x20)
	uartFSMStatus  = uintptr(0x70)
	uartClock      = uintptr(0x88)
	uartRegUpdate  = uintptr(0x98)

	uartBusClockEnable = uint32(1)
	uartReset          = uint32(1 << 1)
	uartClockSource    = uint32(3 << 20) // 40 MHz crystal
	uartSourceEnable   = uint32(1 << 22)
	uartTXClockEnable  = uint32(1 << 24)
	uartRXClockEnable  = uint32(1 << 25)
	uartEightDataBits  = uint32(3 << 2)
	uartOneStopBit     = uint32(1 << 4)
	uartMemoryClock    = uint32(1 << 20)
	uartTXFIFOReset    = uint32(1 << 23)
	uartTXFIFOCount    = uint32(0xff << 16)
	uartTXState        = uint32(0x0f << 4)

	uartCrystalHz = uint32(40000000)
	uartFIFOSize  = uint32(128)
)

// UART is one ESP32-C6 high-performance UART configured as a transmitter.
type UART struct {
	base         uintptr
	pcrConfig    uintptr
	pcrClock     uintptr
	outputSignal uint32
	tx           *Pin
	configured   bool
}

// NewUART1 constructs the UART1 transmitter routed to tx. UART0 is left alone
// for firmware and bootloader compatibility.
func NewUART1(tx *Pin) UART {
	return UART{
		base:         uartOneBase,
		pcrConfig:    uartOnePCRConfig,
		pcrClock:     uartOnePCRClock,
		outputSignal: uartOneOutputSignal,
		tx:           tx,
	}
}

func (u *UART) update() {
	mmio.Store32(u.base+uartRegUpdate, 1)
	for mmio.Load32(u.base+uartRegUpdate)&1 != 0 {
	}
}

// Configure initializes UART1 for 8N1 transmission using the C6's fixed
// 40 MHz crystal clock. The two-stage divider follows Espressif's low-level
// driver so low rates such as MIDI's 31,250 baud remain exact.
func (u *UART) Configure(baud uint32) error {
	if baud == 0 || baud > 1000000 {
		return uart.ErrInvalidBaud
	}
	maximumDividerProduct := uint32(4095) * baud
	sourceDivider := (uartCrystalHz + maximumDividerProduct - 1) / maximumDividerProduct
	if sourceDivider == 0 || sourceDivider > 256 {
		return uart.ErrInvalidBaud
	}
	divider16 := (uartCrystalHz * 16) / (baud * sourceDivider)
	dividerInteger := divider16 >> 4
	if dividerInteger == 0 || dividerInteger > 4095 {
		return uart.ErrInvalidBaud
	}

	clock := mmio.Load32(u.pcrConfig) | uartBusClockEnable
	mmio.Store32(u.pcrConfig, clock|uartReset)
	mmio.Store32(u.pcrConfig, clock&^uartReset)
	mmio.Store32(
		u.pcrClock,
		(sourceDivider-1)<<12|uartClockSource|uartSourceEnable,
	)
	mmio.Store32(u.base+uartClock, uartTXClockEnable|uartRXClockEnable)
	mmio.Store32(u.base+uartClockDiv, dividerInteger|(divider16&0x0f)<<20)
	mmio.Store32(u.base+uartConfigZero, uartMemoryClock|uartEightDataBits|uartOneStopBit)
	u.update()

	config := mmio.Load32(u.base + uartConfigZero)
	mmio.Store32(u.base+uartConfigZero, config|uartTXFIFOReset)
	u.update()
	mmio.Store32(u.base+uartConfigZero, config&^uartTXFIFOReset)
	u.update()

	if err := u.tx.ConfigureOutputSignal(u.outputSignal); err != nil {
		return err
	}
	u.configured = true
	return nil
}

// Write places bytes into the UART transmit FIFO. Each store is deliberately
// 32-bit, as required by the ESP32-C6 FIFO register.
func (u *UART) Write(data []byte) (int, error) {
	if !u.configured {
		return 0, uart.ErrNotConfigured
	}
	for index := 0; index < len(data); index++ {
		for (mmio.Load32(u.base+uartStatus)&uartTXFIFOCount)>>16 >= uartFIFOSize {
		}
		mmio.Store32(u.base+uartFIFO, uint32(data[index]))
	}
	return len(data), nil
}

// Idle reports whether both the transmit FIFO and serializer are empty.
func (u *UART) Idle() bool {
	return mmio.Load32(u.base+uartStatus)&uartTXFIFOCount == 0 &&
		mmio.Load32(u.base+uartFSMStatus)&uartTXState == 0
}

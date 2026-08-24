package esp32c6

import "testing"

func TestUARTOneRegisterMapAndMidiDivider(t *testing.T) {
	controller := NewUART1(GPIO(2))
	if controller.base != 0x60001000 || controller.pcrConfig != 0x6009600c || controller.pcrClock != 0x60096010 {
		t.Fatalf("UART1 registers = %#x/%#x/%#x", controller.base, controller.pcrConfig, controller.pcrClock)
	}
	if controller.outputSignal != 9 {
		t.Fatalf("UART1 TX output signal = %d", controller.outputSignal)
	}
	baud := uint32(31250)
	maximumDividerProduct := uint32(4095) * baud
	sourceDivider := (uartCrystalHz + maximumDividerProduct - 1) / maximumDividerProduct
	divider16 := (uartCrystalHz * 16) / (baud * sourceDivider)
	if sourceDivider != 1 || divider16>>4 != 1280 || divider16&15 != 0 {
		t.Fatalf("MIDI divider = source %d, integer %d, fraction %d", sourceDivider, divider16>>4, divider16&15)
	}
}

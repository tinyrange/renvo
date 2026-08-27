//go:build m5nanoc6

package uart_test

import (
	"renvo.dev/device/board"
	"renvo.dev/device/esp32c6"
	"renvo.dev/device/uart"
)

func ExampleTX_Write() {
	serial := uart.New(board.GroveUART, 115_200)
	_, _ = serial.Write([]byte("ready\r\n"))
}

func ExampleNew() {
	serial := uart.New(board.GroveUART, 9_600)
	_, _ = serial.Write([]byte("AT\r\n"))
}

// DefinePort is used by board packages to hide the selected controller and pin.
func ExampleDefinePort() {
	controller := esp32c6.NewUART1(esp32c6.GPIO(2))
	port := uart.DefinePort(&controller)
	serial := uart.New(port, 115_200)
	_, _ = serial.Write([]byte("ready\r\n"))
}

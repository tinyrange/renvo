//go:build m5nanoc6

package esp32c6_test

import (
	"renvo.dev/device/esp32c6"
	"renvo.dev/device/gpio"
)

func ExamplePin_Configure() {
	interrupt := esp32c6.GPIO(2)
	_ = interrupt.Configure(gpio.Config{Direction: gpio.Input, Pull: gpio.PullUp})
}

func ExamplePin_Set() {
	enable := esp32c6.GPIO(7)
	_ = enable.Configure(gpio.Config{Direction: gpio.Output})
	enable.Set(true)
}

func ExamplePin_Get() {
	interrupt := esp32c6.GPIO(2)
	_ = interrupt.Configure(gpio.Config{Direction: gpio.Input})
	asserted := !interrupt.Get()
	_ = asserted
}

func ExamplePin_ConfigureOpenDrain() {
	data := esp32c6.GPIO(2)
	_ = data.ConfigureOpenDrain()
	data.Release()
}

func ExamplePin_PullLow() {
	data := esp32c6.GPIO(2)
	_ = data.ConfigureOpenDrain()
	data.PullLow()
}

func ExamplePin_Release() {
	data := esp32c6.GPIO(2)
	_ = data.ConfigureOpenDrain()
	data.Release()
	if data.High() {
		// No device is holding the open-drain line low.
	}
}

func ExampleRandom_Uint32() {
	random := esp32c6.Random{}
	identifier := random.Uint32()
	_ = identifier
}

func ExampleSystemTimer_DelayMicroseconds() {
	timer := esp32c6.SystemTimer{}
	timer.DelayMicroseconds(10)
}

func ExampleNewUART1() {
	controller := esp32c6.NewUART1(esp32c6.GPIO(2))
	_ = controller.Configure(115_200)
}

func ExampleUART_Write() {
	controller := esp32c6.NewUART1(esp32c6.GPIO(2))
	_ = controller.Configure(115_200)
	_, _ = controller.Write([]byte("ready\r\n"))
}

func ExampleNewWS2812() {
	strip := esp32c6.NewWS2812(esp32c6.GPIO(20), esp32c6.GPIO(19))
	strip.Set(32, 0, 0)
}

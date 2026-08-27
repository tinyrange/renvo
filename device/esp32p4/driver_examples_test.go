//go:build m5tab5

package esp32p4_test

import (
	"renvo.dev/device/esp32p4"
	"renvo.dev/device/gpio"
)

func ExamplePin_Configure() {
	interrupt := esp32p4.GPIO(23)
	_ = interrupt.Configure(gpio.Config{Direction: gpio.Input, Pull: gpio.PullUp})
}

func ExamplePin_ConfigureOpenDrain() {
	data := esp32p4.GPIO(53)
	_ = data.ConfigureOpenDrain()
	data.Release()
}

func ExamplePin_Set() {
	enable := esp32p4.GPIO(42)
	_ = enable.Configure(gpio.Config{Direction: gpio.Output})
	enable.Set(true)
}

func ExamplePin_Get() {
	interrupt := esp32p4.GPIO(23)
	_ = interrupt.Configure(gpio.Config{Direction: gpio.Input})
	asserted := !interrupt.Get()
	_ = asserted
}

func ExampleSystemTimer_DelayMicroseconds() {
	timer := esp32p4.SystemTimer{}
	timer.DelayMicroseconds(10)
}

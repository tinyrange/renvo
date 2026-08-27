//go:build m5sticks3

package esp32s3_test

import (
	"renvo.dev/device/esp32s3"
	"renvo.dev/device/gpio"
)

func ExamplePin_Configure() {
	interrupt := esp32s3.GPIO(10)
	_ = interrupt.Configure(gpio.Config{Direction: gpio.Input, Pull: gpio.PullUp})
}

func ExamplePin_ConfigureOpenDrain() {
	data := esp32s3.GPIO(1)
	_ = data.ConfigureOpenDrain()
	data.Release()
}

func ExamplePin_Set() {
	enable := esp32s3.GPIO(8)
	_ = enable.Configure(gpio.Config{Direction: gpio.Output})
	enable.Set(true)
}

func ExampleRandom_Uint32() {
	random := esp32s3.Random{}
	value := random.Uint32()
	_ = value
}

func ExampleSystemTimer_DelayMilliseconds() {
	timer := esp32s3.SystemTimer{}
	timer.DelayMilliseconds(10)
}

func ExampleNewWS2812() {
	strip := esp32s3.NewWS2812(esp32s3.GPIO(35), esp32s3.GPIO(34))
	strip.Set(0, 32, 0)
}

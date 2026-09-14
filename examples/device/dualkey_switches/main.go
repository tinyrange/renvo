// Read both Chain DualKey switches with independent debounce and LED feedback.
package main

import (
	"fmt"
	"renvo.dev/device/esp32s3"
	"renvo.dev/device/gpio"
	"renvo.dev/device/input/button"
	"renvo.dev/device/ws2812"
	"renvo.dev/internal/arena"
)

func main() {
	key1, key2 := esp32s3.GPIO(0), esp32s3.GPIO(17)
	key1.Configure(gpio.Config{Direction: gpio.Input, Pull: gpio.PullUp})
	key2.Configure(gpio.Config{Direction: gpio.Input, Pull: gpio.PullUp})
	var timer esp32s3.SystemTimer
	keys := [2]button.Debouncer{{DelayMS: 20}, {DelayMS: 20}}
	var presses [2]uint32
	// DualKey's LED supply enable is active low.
	ledPower := esp32s3.GPIO(40)
	ledPower.Set(false)
	ledPower.Configure(gpio.Config{Direction: gpio.Output})
	timer.DelayMilliseconds(10)
	leds := ws2812.New(esp32s3.GPIO(21), nil)
	pixels := [2]ws2812.RGB{{Blue: 24}, {Blue: 24}}
	leds.SetPixels(pixels[:])
	mark := arena.Mark()
	last := timer.Ticks()
	var now, remainder, report uint32
	print("RENVO CHAIN DUALKEY: GPIO0 / GPIO17; debounce 20 ms\n")
	for {
		ticks := timer.Ticks()
		remainder += ticks - last
		last = ticks
		now += remainder / 16000
		remainder %= 16000
		raw := [2]bool{!key1.Get(), !key2.Get()}
		changed := false
		for i := 0; i < 2; i++ {
			key := &keys[i]
			if key.Update(raw[i], now) {
				changed = true
				state := "UP"
				// The daisy-chain reaches the LED under Key2 before Key1.
				pixel := 1 - i
				pixels[pixel] = ws2812.RGB{Blue: 24}
				if key.Pressed {
					state = "DOWN"
					presses[i]++
					pixels[pixel] = ws2812.RGB{Green: 24}
				}
				fmt.Printf("KEY%d %s time=%d presses=%d\n", i+1, state, now, presses[i])
			}
		}
		if changed {
			leds.SetPixels(pixels[:])
		}
		if now-report >= 2000 {
			report = now
			fmt.Printf("KEY1:%d KEY2:%d PRESS1:%d PRESS2:%d\n", level(keys[0].Pressed), level(keys[1].Pressed), presses[0], presses[1])
		}
		arena.Rewind(mark)
		timer.DelayMilliseconds(1)
	}
}

func level(v bool) int {
	if v {
		return 1
	}
	return 0
}

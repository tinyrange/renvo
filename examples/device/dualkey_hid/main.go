// Type A from the left DualKey switch and B from the right switch.
package main

import (
	"renvo.dev/device/esp32s3"
	"renvo.dev/device/esp32s3/usbhid"
	"renvo.dev/device/gpio"
	"renvo.dev/device/input/button"
	"renvo.dev/device/ws2812"
	"renvo.dev/internal/arena"
)

func main() {
	key1, key2 := esp32s3.GPIO(0), esp32s3.GPIO(17)
	key1.Configure(gpio.Config{Direction: gpio.Input, Pull: gpio.PullUp})
	key2.Configure(gpio.Config{Direction: gpio.Input, Pull: gpio.PullUp})
	power := esp32s3.GPIO(40)
	power.Set(false)
	power.Configure(gpio.Config{Direction: gpio.Output})
	var timer esp32s3.SystemTimer
	timer.DelayMilliseconds(10)
	leds := ws2812.New(esp32s3.GPIO(21), nil)
	pixels := [2]ws2812.RGB{{Blue: 24}, {Blue: 24}}
	leds.SetPixels(pixels[:])
	print("RENVO DUALKEY HID: USB takeover in 5 seconds; hold both keys 5 seconds to return to flashing mode\n")
	timer.DelayMilliseconds(5000)
	// Reuse this board's factory VID/PID for local development only.
	if !usbhid.Start(usbhid.Config{
		VendorID: 0x303a, ProductID: 0x8000,
		Manufacturer: "Renvo", Product: "Renvo DualKey", Serial: "DualKey-dev",
		Keyboard: true,
	}) {
		pixels[0] = ws2812.RGB{Red: 24}
		pixels[1] = pixels[0]
		leds.SetPixels(pixels[:])
		return
	}
	keys := [2]button.Debouncer{{DelayMS: 20}, {DelayMS: 20}}
	mark := arena.Mark()
	last := timer.Ticks()
	var now, remainder, bothSince uint32
	for {
		ticks := timer.Ticks()
		remainder += ticks - last
		last = ticks
		now += remainder / 16000
		remainder %= 16000
		raw := [2]bool{!key1.Get(), !key2.Get()}
		changed := false
		var report byte
		for i := 0; i < 2; i++ {
			if keys[i].Update(raw[i], now) {
				changed = true
				pixels[1-i] = ws2812.RGB{Blue: 24}
				if keys[i].Pressed {
					pixels[1-i] = ws2812.RGB{Green: 24}
				}
			}
			if keys[i].Pressed {
				// Left is Key2/GPIO17; right is Key1/GPIO0 on this board.
				// Keyboard report bit 0 is A and bit 1 is B.
				report |= byte(1) << uint(1-i)
			}
		}
		usbhid.Poll(report)
		if changed {
			leds.SetPixels(pixels[:])
		}
		if report != 3 {
			bothSince = now
		} else if now-bothSince >= 5000 {
			// Deliver a release before disconnecting the keyboard.
			for i := 0; i < 30; i++ {
				usbhid.Poll(0)
				timer.DelayMilliseconds(1)
			}
			usbhid.Stop()
			pixels[0] = ws2812.RGB{Red: 24, Blue: 24}
			pixels[1] = pixels[0]
			leds.SetPixels(pixels[:])
			for {
				timer.DelayMilliseconds(1000)
			}
		}
		arena.Rewind(mark)
		timer.DelayMilliseconds(1)
	}
}

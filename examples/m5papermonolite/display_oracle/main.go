package main

import (
	"renvo.dev/device/board"
	"renvo.dev/device/display/ssd1677"
)

var plane1 ssd1677.Monochrome

type grayPattern struct{}

func (grayPattern) FillGrayRow(plane, row int, destination []byte) error {
	for column := range destination {
		high := byte(0)
		if plane == 0 {
			high = 0xff
		}
		if row < ssd1677.Height/2 {
			if column < ssd1677.BytesPerRow/2 {
				destination[column] = 0x00
			} else {
				destination[column] = high
			}
		} else {
			if column < ssd1677.BytesPerRow/2 {
				if plane == 0 {
					destination[column] = 0x00
				} else {
					destination[column] = 0xff
				}
			} else {
				destination[column] = 0xff
			}
		}
	}
	return nil
}

func fail(message string) {
	print(message)
	_ = board.Display.Shutdown()
	for {
	}
}

func setRectangle(frame *ssd1677.Monochrome, x, y, width, height int, white bool) {
	for row := y; row < y+height; row++ {
		for column := x; column < x+width; column++ {
			frame.Set(column, row, white)
		}
	}
}

func makeMonochromePattern() {
	plane1.Fill(true)
	setRectangle(&plane1, 0, 0, ssd1677.Width, 8, false)
	setRectangle(&plane1, 0, ssd1677.Height-8, ssd1677.Width, 8, false)
	setRectangle(&plane1, 0, 0, 8, ssd1677.Height, false)
	setRectangle(&plane1, ssd1677.Width-8, 0, 8, ssd1677.Height, false)
	setRectangle(&plane1, ssd1677.Width/2-4, 0, 8, ssd1677.Height, false)
	setRectangle(&plane1, 0, ssd1677.Height/2-4, ssd1677.Width, 8, false)
}

func main() {
	if board.Initialize() != nil {
		fail("RENVO PAPERMONO-LITE PHASE3 BOARD FAIL\n")
	}
	board.Clock.DelayMilliseconds(500)
	makeMonochromePattern()
	if err := board.Display.FullMonochrome(plane1[:]); err != nil {
		fail("RENVO PAPERMONO-LITE PHASE3 FULL MONO FAIL\n")
	}
	print("RENVO PAPERMONO-LITE PHASE3 FULL MONO PASS\n")

	for update := 0; update < 10; update++ {
		white := update&1 != 0
		setRectangle(&plane1, 24, 24, 96, 96, white)
		if err := board.Display.PartialMonochrome(plane1[:]); err != nil {
			fail("RENVO PAPERMONO-LITE PHASE3 PARTIAL FAIL\n")
		}
		print("RENVO PAPERMONO-LITE PHASE3 PARTIAL PASS\n")
	}
	// The eleventh request must be promoted to a full baseline recovery.
	setRectangle(&plane1, 24, 24, 96, 96, false)
	if err := board.Display.PartialMonochrome(plane1[:]); err != nil {
		fail("RENVO PAPERMONO-LITE PHASE3 RECOVERY FAIL\n")
	}
	print("RENVO PAPERMONO-LITE PHASE3 RECOVERY PASS\n")

	if err := board.Display.FullGrayStream(grayPattern{}); err != nil {
		fail("RENVO PAPERMONO-LITE PHASE3 FOUR GRAY FAIL\n")
	}
	print("RENVO PAPERMONO-LITE PHASE3 FOUR GRAY PASS\n")
	if err := board.Display.Shutdown(); err != nil {
		fail("RENVO PAPERMONO-LITE PHASE3 SHUTDOWN FAIL\n")
	}
	print("RENVO PAPERMONO-LITE PHASE3 SHUTDOWN PASS\n")
	for {
	}
}

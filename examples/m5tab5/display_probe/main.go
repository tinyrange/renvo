package main

import "renvo.dev/examples/m5tab5/board"

func main() {
	if !board.InitFramebuffer() {
		for {
			print("TAB5 DISPLAY INIT FAIL\n")
			for delay := 0; delay < 20000000; delay++ {
			}
		}
	}
	if !board.InitTouch() {
		for {
			board.Refresh()
			print("TAB5 TOUCH INIT FAIL\n")
			for delay := 0; delay < 20000000; delay++ {
			}
		}
	}
	count := 0
	for {
		board.Refresh()
		count++
		if count == 1000000 {
			print("TAB5 PSRAM FRAME + TOUCH PASS\n")
			count = 0
		}
	}
}

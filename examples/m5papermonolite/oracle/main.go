package main

import "renvo.dev/device/board"

func main() {
	if board.Initialize() == nil {
		// Native USB re-enumerates after reset. Give the host monitor time to
		// reopen so this one-shot oracle cannot disappear during enumeration.
		board.Clock.DelayMilliseconds(500)
		print("RENVO PAPERMONO-LITE PASS\n")
	}
	for {
	}
}

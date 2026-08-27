package main

import "renvo.dev/device/bios"

//renvo:compile -t freestanding/amd64 longModeMain
var longModeEntry uintptr

// sharedMessage is deliberately reachable from both entrypoints. It is emitted
// once by the 8086 unit and independently by the amd64 unit.
func sharedMessage() string { return "shared function" }

func debugLine(text string) {
	for i := 0; i < len(text); i++ {
		bios.Out8(0xe9, text[i])
	}
	bios.Out8(0xe9, '\n')
}

func longModeMain() {
	print("LONG MODE: ", sharedMessage(), "\n")
}

func main() {
	debugLine("REAL MODE: " + sharedMessage())
	bios.EnterLongMode(longModeEntry)
}

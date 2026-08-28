package terminal_test

import (
	"fmt"

	"renvo.dev/device/terminal"
)

// A model-only terminal can be used without a display and attached to one later.
func ExampleTerminal_WriteString() {
	state := terminal.New(80, 25, 500)
	state.WriteString("ready\r\n> ")
	row, column := state.Cursor()
	fmt.Printf("cursor=%d,%d\n", row, column)
	// Output: cursor=1,2
}

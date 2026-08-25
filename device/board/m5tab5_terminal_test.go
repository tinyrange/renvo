//go:build m5tab5

package board

import "testing"

func TestDefaultTerminalOptionsFitTab5(t *testing.T) {
	options := defaultTerminalOptions()
	if options.Font == nil || options.CellWidth != 18 || options.CellHeight != 30 || options.Baseline != 21 {
		t.Fatalf("terminal font geometry = %#v", options)
	}
	if !options.TouchKeyboard || options.Pointer != &Touch || options.Clock != &Display {
		t.Fatalf("terminal board capabilities = %#v", options)
	}
}

func TestTickTerminalWithoutConsoleStopsLoop(t *testing.T) {
	previous := Console
	Console = nil
	defer func() { Console = previous }()
	if TickTerminal() {
		t.Fatal("terminal tick continued without a console")
	}
}

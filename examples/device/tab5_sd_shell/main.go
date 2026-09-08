package main

import (
	"renvo.dev/device/board"
	"renvo.dev/device/fat32"
	"renvo.dev/device/i2c"
	"renvo.dev/device/input/tab5keyboard"
	"renvo.dev/device/shell"
	"renvo.dev/device/terminal"
	"renvo.dev/std/fmt"
	"renvo.dev/std/graphics"
)

func main() {
	board.Display.SetLandscape(true)
	console, err := terminal.Start(&board.Display, terminal.Options{Scrollback: 128, Font: graphics.NewBuiltinFont(2), CellWidth: 12, CellHeight: 20, Baseline: 14, FlushPolicy: terminal.FlushLine, Clock: &board.Display, Pointer: &board.Touch})
	if err != nil {
		print("terminal failed\r\n")
		return
	}
	fmt.Fprint(console, "\r\nRENVO SD SHELL - FAT32 / 4-bit SDMMC DMA\r\nMounting card (no formatting)...\r\n")
	card, err := board.OpenSD()
	if err != nil {
		fmt.Fprintf(console, "SD: %s\r\n", err)
		console.Flush()
		return
	}
	volume, err := fat32.Mount(card)
	if err != nil {
		fmt.Fprintf(console, "FAT32: %s\r\n", err)
		console.Flush()
		return
	}
	keyboard := tab5keyboard.New(i2c.New(board.ExtPort1()))
	if err = keyboard.Initialize(tab5keyboard.Character); err != nil {
		fmt.Fprintf(console, "Keyboard: %s\r\n", err)
		console.Flush()
		return
	}
	fmt.Fprintf(console, "%d sectors. Type help. Writes occur only on explicit commands.\r\n", card.Blocks())
	sh := shell.New(volume, console)
	sh.Prompt()
	console.Flush()
	line := ""
	history := []string{}
	historyIndex := 0
	escape := 0
	var input [16]byte
	for console.Tick(16 * terminal.Millisecond) {
		for batch := 0; batch < 32; batch++ {
			event, ok, readErr := keyboard.NextCharacter()
			if readErr != nil {
				fmt.Fprintf(console, "\r\nKeyboard: %s\r\n", readErr)
				break
			}
			if !ok {
				break
			}
			n := event.TerminalBytes(input[:])
			for i := 0; i < n; i++ {
				c := input[i]
				if escape == 1 {
					if c == '[' {
						escape = 2
					} else {
						escape = 0
					}
					continue
				}
				if escape == 2 {
					escape = 0
					if c == 'A' && historyIndex > 0 {
						historyIndex--
					} else if c == 'B' && historyIndex < len(history) {
						historyIndex++
					} else {
						continue
					}
					line = ""
					if historyIndex < len(history) {
						line = history[historyIndex]
					}
					fmt.Fprint(console, "\r\x1b[2K")
					sh.Prompt()
					fmt.Fprint(console, line)
					continue
				}
				switch c {
				case 27:
					escape = 1
				case 15:
					board.Display.SetLandscape(!board.Display.Landscape)
					console.Flush()
					fmt.Fprint(console, "\r\x1b[2K")
					sh.Prompt()
					fmt.Fprint(console, line)
				case 12:
					console.Reset()
					sh.Prompt()
					fmt.Fprint(console, line)
				case 3:
					fmt.Fprint(console, "^C\r\n")
					line = ""
					sh.Prompt()
				case '\r':
					fmt.Fprint(console, "\r\n")
					if line != "" {
						if len(history) == 16 {
							history = history[1:]
						}
						history = append(history, line)
					}
					sh.Execute(line)
					line = ""
					historyIndex = len(history)
					sh.Prompt()
				case '\n': // Keyboard Enter supplies CRLF; execute only once.
				case 8, 127:
					if len(line) > 0 {
						line = line[:len(line)-1]
						fmt.Fprint(console, "\b \b")
					}
				default:
					if c >= 32 && c < 127 && len(line) < 256 {
						line += string(input[i : i+1])
						console.WriteByte(c)
					}
				}
			}
		}
		console.Flush()
	}
	volume.Sync()
}

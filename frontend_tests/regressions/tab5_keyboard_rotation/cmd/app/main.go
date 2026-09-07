package main

import (
	"renvo.dev/device/input/tab5keyboard"
	"renvo.dev/device/terminal"
	"renvo.dev/forms"
	"renvo.dev/std/graphics"
)

func check(ok bool) {
	if !ok {
		panic("Tab5 rotation regression")
	}
}

type bus struct {
	reg  [256]byte
	size int
}

func (b *bus) Tx(address uint16, write, read []byte) error {
	check(address == 0x6d)
	reg := int(write[0])
	if len(read) > 0 {
		copy(read, b.reg[reg:reg+len(read)])
		b.size = len(read)
	} else {
		copy(b.reg[reg:reg+len(write)-1], write[1:])
	}
	return nil
}
func (*bus) DelayMilliseconds(uint32) {}

type display struct{ surface *graphics.Surface }

func (d *display) InitializeTerminal() (*graphics.Surface, bool) { return d.surface, true }
func (*display) PresentTerminal(*graphics.Surface) bool          { return true }

func main() {
	b := &bus{}
	d := tab5keyboard.New(b)
	check(d.Initialize(tab5keyboard.Character) == nil)
	b.reg[0x40] = 10
	b.reg[0x50] = 4
	name := "BACKSPACE"
	for i := 0; i < len(name); i++ {
		b.reg[0x51+i] = name[i]
	}
	e, ok, err := d.NextCharacter()
	check(err == nil && ok && e.Length == 9 && b.size == 10)
	var input [5]byte
	n := e.TerminalBytes(input[:])
	check(n == 2 && input[0] == 27 && input[1] == 8)
	check(d.SetRGB(1, 10, 20, 30) == nil && b.reg[0x64] == 30 && b.reg[0x66] == 10)
	b.reg[0x20] = 0xcd
	k, ok, err := d.NextKey()
	check(err == nil && ok && k.Row == 4 && k.Column == 13 && k.Pressed)
	s := graphics.NewRotatedSurface(60, 100, graphics.PixelRGB565, graphics.Rotation90)
	check(s.Width == 100 && s.Height == 60 && s.Stride == 120 && len(s.Pixels) == 12000)
	s.Clear(graphics.Black)
	s.FillRect(graphics.R(1, 2, 3, 4), graphics.White)
	offset := (100-1-1)*120 + 2*2
	check(s.Pixels[offset] == 255 && s.Pixels[offset+1] == 255 && s.Pixels[0] == 0)
	check(s.CopyPixels(1, 1, 1, 2, 3, 4))
	offset = (100-1-1)*120 + 1*2
	check(s.Pixels[offset] == 255)
	s.DrawText(graphics.NewBuiltinFont(1), graphics.Point{X: 5, Y: 20}, "Tab5", graphics.White)
	var form forms.Form
	form.Initialize(100, 60)
	check(form.RotateSurface(s, graphics.Rotation0))
	w, h := form.Size()
	check(w == 60 && h == 100 && form.Paint(s))
	display := &display{surface: s}
	console, err := terminal.Start(display, terminal.Options{Font: graphics.NewBuiltinFont(1), CellWidth: 6, CellHeight: 10, FlushPolicy: terminal.FlushManual, Scrollback: 8})
	check(err == nil)
	for i := 0; i < 4; i++ {
		rotation := graphics.Rotation90
		if i%2 != 0 {
			rotation = graphics.Rotation0
		}
		check(s.SetRotation(rotation))
		check(console.Flush() && console.Columns() == s.Width/6 && console.Rows() == s.Height/10)
		for line := 0; line < 14; line++ {
			console.WriteString("hello\r\n")
			check(console.Flush())
		}
	}
	console.Stop()
	print("PASS\n")
}

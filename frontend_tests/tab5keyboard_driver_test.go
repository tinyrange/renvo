package frontend_tests

import (
	"bytes"
	"errors"
	"renvo.dev/device/input/tab5keyboard"
	"testing"
)

type keyboardTransfer struct {
	address     uint16
	write, read []byte
}
type keyboardBus struct {
	t         *testing.T
	transfers []keyboardTransfer
	delay     uint32
	fail      error
}

func (b *keyboardBus) Tx(address uint16, write, read []byte) error {
	b.t.Helper()
	if len(b.transfers) == 0 {
		b.t.Fatal("unexpected transaction", address, write, len(read))
	}
	next := b.transfers[0]
	b.transfers = b.transfers[1:]
	if address != next.address || !bytes.Equal(write, next.write) || len(read) != len(next.read) {
		b.t.Fatalf("transaction %x %x read=%d; want %+v", address, write, len(read), next)
	}
	copy(read, next.read)
	return b.fail
}
func (b *keyboardBus) DelayMilliseconds(ms uint32) { b.delay += ms }
func (b *keyboardBus) check() {
	b.t.Helper()
	if len(b.transfers) != 0 {
		b.t.Fatal("missing transfers", b.transfers)
	}
}

func TestTab5KeyboardConfiguration(t *testing.T) {
	b := &keyboardBus{t: t, transfers: []keyboardTransfer{
		{0x6d, []byte{0xfe}, []byte{1}}, {0x6d, []byte{0x10, 2}, nil}, {0x6d, []byte{2, 0}, nil},
		{0x6d, []byte{0, 7}, nil}, {0x6d, []byte{1, 0}, nil}, {0x6d, []byte{3, 100}, nil},
		{0x6d, []byte{0x11, 1}, nil}, {0x6d, []byte{0x60, 30, 20, 10}, nil}, {0x6d, []byte{0x64, 60, 50, 40}, nil},
		{0x6d, []byte{0x60}, []byte{3, 2, 1}}, {0x6d, []byte{0x64}, []byte{6, 5, 4}},
		{0x6d, []byte{0xff, 0x70}, nil}, {0x70, []byte{0xfe}, []byte{1}},
	}}
	d := tab5keyboard.New(b)
	for _, err := range []error{d.Initialize(tab5keyboard.Character), d.SetInterruptEnable(7), d.ClearInterrupt(), d.SetBrightness(100), d.SetRGBMode(tab5keyboard.RGBCustom), d.SetRGB(0, 10, 20, 30), d.SetRGB(1, 40, 50, 60)} {
		if err != nil {
			t.Fatal(err)
		}
	}
	r, g, blue, err := d.RGB(0)
	if err != nil || r != 1 || g != 2 || blue != 3 {
		t.Fatal("RGB0", r, g, blue, err)
	}
	r, g, blue, err = d.RGB(1)
	if err != nil || r != 4 || g != 5 || blue != 6 {
		t.Fatal("RGB1", r, g, blue, err)
	}
	if err := d.SetAddress(0x70); err != nil || d.Address() != 0x70 || b.delay != 50 {
		t.Fatal("address", err)
	}
	if version, err := d.FirmwareVersion(); err != nil || version != 1 {
		t.Fatal(version, err)
	}
	b.check()
	for _, err := range []error{d.SetAddress(0x07), d.SetAddress(0x78), d.SetBrightness(101), d.SetMode(3), d.SetRGBMode(2), d.SetRGB(2, 0, 0, 0), d.SetInterruptEnable(8)} {
		if err != tab5keyboard.ErrRange {
			t.Fatal("validation", err)
		}
	}
	if _, err := tab5keyboard.NewAt(b, 0x80); err != tab5keyboard.ErrRange {
		t.Fatal("constructor range")
	}
}

func TestTab5KeyboardRegisters(t *testing.T) {
	reads := []struct {
		reg  byte
		read func(*tab5keyboard.Device) (byte, error)
	}{
		{0, func(d *tab5keyboard.Device) (byte, error) { return d.InterruptEnable() }},
		{1, func(d *tab5keyboard.Device) (byte, error) { return d.InterruptStatus() }},
		{2, func(d *tab5keyboard.Device) (byte, error) { return d.EventCount() }},
		{3, func(d *tab5keyboard.Device) (byte, error) { return d.Brightness() }},
		{0x10, func(d *tab5keyboard.Device) (byte, error) { v, e := d.Mode(); return byte(v), e }},
		{0x11, func(d *tab5keyboard.Device) (byte, error) { v, e := d.RGBMode(); return byte(v), e }},
	}
	for _, tc := range reads {
		b := &keyboardBus{t: t, transfers: []keyboardTransfer{{0x6d, []byte{tc.reg}, []byte{2}}}}
		v, err := tc.read(tab5keyboard.New(b))
		if v != 2 || err != nil {
			t.Fatal(tc.reg, v, err)
		}
		b.check()
	}
}

func TestTab5KeyboardEvents(t *testing.T) {
	b := &keyboardBus{t: t, transfers: []keyboardTransfer{
		{0x6d, []byte{0x20}, []byte{0xcd}}, {0x6d, []byte{0x20}, []byte{0x4d}}, {0x6d, []byte{0x20}, []byte{0xff}},
		{0x6d, []byte{0x30}, []byte{5, 4}}, {0x6d, []byte{0x30}, []byte{0, 0}}, {0x6d, []byte{0x30}, []byte{255, 255}},
		{0x6d, []byte{0x40}, []byte{10}}, {0x6d, []byte{0x50}, []byte{5, 'b', 'a', 'c', 'k', 's', 'p', 'a', 'c', 'e'}},
		{0x6d, []byte{0x40}, []byte{0}},
	}}
	d := tab5keyboard.New(b)
	e, ok, err := d.NextKey()
	if err != nil || !ok || e.Row != 4 || e.Column != 13 || !e.Pressed {
		t.Fatal(e, ok, err)
	}
	e, ok, err = d.NextKey()
	if err != nil || !ok || e.Pressed {
		t.Fatal(e, ok, err)
	}
	_, ok, err = d.NextKey()
	if err != nil || ok {
		t.Fatal("normal empty")
	}
	h, ok, err := d.NextHID()
	if err != nil || !ok || h.Modifier != 5 || h.Keycode != 4 {
		t.Fatal(h, ok, err)
	}
	h, ok, err = d.NextHID()
	if err != nil || !ok || h.Keycode != 0 {
		t.Fatal("release", h, ok, err)
	}
	_, ok, err = d.NextHID()
	if err != nil || ok {
		t.Fatal("HID empty")
	}
	c, ok, err := d.NextCharacter()
	if err != nil || !ok || c.Modifier != 5 || c.Length != 9 || string(c.Text[:]) != "backspace" {
		t.Fatal(c, ok, err)
	}
	_, ok, err = d.NextCharacter()
	if err != nil || ok {
		t.Fatal("char empty")
	}
	b.check()
	for _, length := range []byte{1, 11, 255} {
		b.transfers = []keyboardTransfer{{0x6d, []byte{0x40}, []byte{length}}}
		_, ok, err = d.NextCharacter()
		if ok || err != tab5keyboard.ErrProtocol {
			t.Fatal("invalid length", length, err)
		}
		b.check()
	}
	for _, value := range []byte{0x50, 0x0e, 0xce} {
		b.transfers = []keyboardTransfer{{0x6d, []byte{0x20}, []byte{value}}}
		_, ok, err = d.NextKey()
		if ok || err != tab5keyboard.ErrProtocol {
			t.Fatal("invalid matrix", value, err)
		}
	}
}

func TestTab5KeyboardFailuresDoNotPublishPartialData(t *testing.T) {
	b := &keyboardBus{t: t, fail: errors.New("NAK"), transfers: []keyboardTransfer{{0x6d, []byte{0xff, 0x71}, nil}, {0x6d, []byte{0x30}, []byte{1, 4}}, {0x6d, []byte{0x40}, []byte{2}}}}
	d := tab5keyboard.New(b)
	if err := d.SetAddress(0x71); err != b.fail || d.Address() != 0x6d || b.delay != 0 {
		t.Fatal("failed readdress")
	}
	h, ok, err := d.NextHID()
	if err != b.fail || ok || h != (tab5keyboard.HIDEvent{}) {
		t.Fatal("partial HID")
	}
	_, ok, err = d.NextCharacter()
	if err != b.fail || ok {
		t.Fatal("partial char length")
	}
	b.transfers = []keyboardTransfer{{0x6d, []byte{0x40}, []byte{2}}, {0x6d, []byte{0x50}, []byte{0, 'a'}}}
	b.fail = nil
	// Separately prove exact minimum packet framing.
	e, ok, err := d.NextCharacter()
	if err != nil || !ok || e.Length != 1 || e.Text[0] != 'a' {
		t.Fatal(e, ok, err)
	}
	b.check()
}

func TestTab5KeyboardTerminalInput(t *testing.T) {
	for _, tc := range []struct {
		text     string
		modifier byte
		want     string
	}{
		{"a", 0, "a"}, {"C", 1, "\x03"}, {"c", 5, "\x1b\x03"}, {" ", 1, "\x00"}, {"?", 1, "\x7f"},
		{"esc", 0, "\x1b"}, {"TAB", 0, "\t"}, {"ENTER", 0, "\r\n"}, {"backspace", 0, "\x08"},
		{"DEL", 4, "\x1b\x1b[3~"}, {"UP", 0, "\x1b[A"}, {"down", 0, "\x1b[B"}, {"left", 0, "\x1b[D"}, {"right", 0, "\x1b[C"}, {"unknown", 0, ""},
	} {
		e := tab5keyboard.CharacterEvent{Modifier: tc.modifier, Length: len(tc.text)}
		copy(e.Text[:], tc.text)
		var dst [5]byte
		n := e.TerminalBytes(dst[:])
		if string(dst[:n]) != tc.want {
			t.Fatalf("%s: %q != %q", tc.text, dst[:n], tc.want)
		}
		if e.TerminalBytes(dst[:4]) != 0 {
			t.Fatal("short buffer accepted")
		}
	}
}

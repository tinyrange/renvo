//go:build linux

package espflash

import "testing"

func TestParseDebugUint32(t *testing.T) {
	tests := []struct {
		text  string
		value uint32
		ok    bool
	}{
		{text: "0", ok: true},
		{text: "1234", value: 1234, ok: true},
		{text: "0x40824100", value: 0x40824100, ok: true},
		{text: "0XFFFFFFFF", value: 0xffffffff, ok: true},
		{text: "", ok: false},
		{text: "0x", ok: false},
		{text: "12z", ok: false},
		{text: "4294967296", ok: false},
		{text: "0x100000000", ok: false},
	}
	for _, test := range tests {
		value, ok := parseDebugUint32(test.text)
		if value != test.value || ok != test.ok {
			t.Fatalf("parseDebugUint32(%q) = %#x, %v; want %#x, %v", test.text, value, ok, test.value, test.ok)
		}
	}
}

func TestParseRegister(t *testing.T) {
	tests := []struct {
		name     string
		register int
		ok       bool
	}{
		{name: "x0", register: 0, ok: true},
		{name: "ra", register: 1, ok: true},
		{name: "sp", register: 2, ok: true},
		{name: "a0", register: 10, ok: true},
		{name: "s11", register: 27, ok: true},
		{name: "x31", register: 31, ok: true},
		{name: "x32", ok: false},
		{name: "pc", ok: false},
	}
	for _, test := range tests {
		register, ok := parseRegister(test.name)
		if register != test.register || ok != test.ok {
			t.Fatalf("parseRegister(%q) = %d, %v; want %d, %v", test.name, register, ok, test.register, test.ok)
		}
	}
}

func TestDebugFormatting(t *testing.T) {
	if got := hex32(0x1234); got != "00001234" {
		t.Fatalf("hex32 = %q", got)
	}
	if got := hexByte(0xaf); got != "af" {
		t.Fatalf("hexByte = %q", got)
	}
	if got := registerLabel(10); got != "x10/a0  " {
		t.Fatalf("registerLabel = %q", got)
	}
}

//go:build linux || darwin

package espflash

import "testing"

func TestFlashSizeFromJEDECID(t *testing.T) {
	tests := []struct {
		name string
		id   uint32
		size int
		ok   bool
	}{
		{name: "winbond 4 MiB", id: 0x1640ef, size: 4 * 1024 * 1024, ok: true},
		{name: "winbond 8 MiB", id: 0x1740ef, size: 8 * 1024 * 1024, ok: true},
		{name: "winbond 16 MiB", id: 0x1840ef, size: 16 * 1024 * 1024, ok: true},
		{name: "alternate 16 MiB capacity code", id: 0x3840c2, size: 16 * 1024 * 1024, ok: true},
		{name: "adesto 16 MiB", id: 0x00091f, size: 16 * 1024 * 1024, ok: true},
		{name: "missing device", id: 0xffffff},
		{name: "zero response", id: 0},
		{name: "unsupported 64 MiB", id: 0x1a40ef},
		{name: "unknown capacity", id: 0x1140ef},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			size, ok := flashSizeFromJEDECID(test.id)
			if size != test.size || ok != test.ok {
				t.Fatalf("flashSizeFromJEDECID(0x%x) = (%d, %v), want (%d, %v)", test.id, size, ok, test.size, test.ok)
			}
		})
	}
}

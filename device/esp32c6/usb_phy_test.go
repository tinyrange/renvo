package esp32c6

import (
	"testing"

	"renvo.dev/device/usb/lowspeed"
)

func TestUSBLine(t *testing.T) {
	tests := []struct {
		dp, dm bool
		line   byte
		valid  bool
	}{
		{line: 0, valid: true},
		{dm: true, line: 1, valid: true},
		{dp: true, line: 2, valid: true},
		{dp: true, dm: true},
	}
	for _, test := range tests {
		line, valid := usbLine(test.dp, test.dm)
		if line != test.line || valid != test.valid {
			t.Fatalf("usbLine(%v, %v) = (%d, %v), want (%d, %v)",
				test.dp, test.dm, line, valid, test.line, test.valid)
		}
	}
}

func TestDecodeRMTPacket(t *testing.T) {
	var waveform [128]byte
	waveCount := lowspeed.EncodeToken(waveform[:], lowspeed.PIDSetup, 0, 0)
	var halves []uint32
	for index := 0; index < waveCount-3; {
		line := waveform[index]
		end := index + 1
		for end < waveCount-3 && waveform[end] == line {
			end++
		}
		half := uint32((end - index) * 53)
		if line == lowspeed.LineK {
			half |= 1 << 15
		}
		halves = append(halves, half)
		index = end
	}
	var words []uint32
	for index := 0; index < len(halves); index += 2 {
		word := halves[index]
		if index+1 < len(halves) {
			word |= halves[index+1] << 16
		}
		words = append(words, word)
	}
	var got [128]byte
	var data [16]byte
	count, _, _ := decodeRMTPacket(got[:], data[:], words)
	if count != waveCount {
		t.Fatalf("count = %d, want %d", count, waveCount)
	}
	for index := 0; index < count; index++ {
		if got[index] != waveform[index] {
			t.Fatalf("state %d = %d, want %d", index, got[index], waveform[index])
		}
	}
}

func TestDecodeRMTHardwareCapture(t *testing.T) {
	words := []uint32{
		0x00368035, 0x00358035, 0x00368035, 0x00a080a0,
		0x0036806a, 0x00368035, 0x00358035, 0x00368035,
		0x00368035, 0x00358035, 0x00368035, 0x0036806a,
		0x014e8035, 0x00358035, 0x00368035, 0x00368035,
		0x003580d5, 0x00368035, 0x003580a0, 0x00368035,
		0x00368035, 0x006b8035, 0x0036809f, 0x00368035,
		0x00358035, 0x00368035, 0x00368035, 0x00358035,
		0x006b8035, 0x00368035, 0x00368035, 0x00358035,
		0x00368035, 0x00368035, 0x00358035, 0x00368035,
		0x00368035, 0x00358035, 0x00368035, 0x00368035,
		0x00358035, 0x00368035, 0x00368035, 0x0036806a,
		0x00358035, 0x00368035, 0x00368035, 0x006b8035,
	}
	var states [128]byte
	var data [16]byte
	if count, _, _ := decodeRMTPacket(states[:], data[:], words); count == 0 {
		t.Fatal("hardware RMT capture did not contain a valid USB packet")
	}
}

func TestEncodeRMTSignalTiming(t *testing.T) {
	var states [128]byte
	count := lowspeed.EncodeHandshake(states[:], lowspeed.PIDAck)
	var words [48]uint32
	wordCount := encodeRMTSignal(words[:], states[:count], true)
	if wordCount == 0 {
		t.Fatal("encodeRMTSignal rejected an ACK")
	}
	total := uint32(0)
	for index := 0; index < wordCount; index++ {
		first := words[index] & 0x7fff
		second := words[index] >> 16 & 0x7fff
		total += first + second
	}
	want := uint32(count/3*160 + count%3*53)
	if total != want {
		t.Fatalf("encoded duration = %d, want %d", total, want)
	}
}

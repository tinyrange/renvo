package esp32c6

import (
	"testing"

	"renvo.dev/device/usb/lowspeed"
)

func TestBuildFullSpeedWaveform(t *testing.T) {
	var states [128]byte
	count := lowspeed.EncodeData(states[:], lowspeed.PIDData1, []byte{0x55})
	var waveform FullSpeedWaveform
	if !waveform.BuildFullSpeed(states[:count]) {
		t.Fatal("BuildFullSpeed rejected a valid DATA1 packet")
	}
	if waveform.Entry() == 0 || waveform.words == 0 {
		t.Fatal("BuildFullSpeed did not publish an entry point")
	}
	stores := 0
	for _, instruction := range waveform.code[:waveform.words] {
		if instruction == rvStoreT4AtT0 || instruction == rvStoreT5AtT0 {
			stores++
		}
	}
	if want := count - 3 + 1; stores != want {
		t.Fatalf("differential stores = %d, want %d", stores, want)
	}
	if waveform.code[waveform.words-1] != rvReturn {
		t.Fatal("waveform does not return through ra")
	}
}

func TestBuildFullSpeedWaveformRejectsInvalidEOP(t *testing.T) {
	var waveform FullSpeedWaveform
	if waveform.BuildFullSpeed([]byte{1, 2, 1}) {
		t.Fatal("BuildFullSpeed accepted a packet without SE0, SE0, J")
	}
}

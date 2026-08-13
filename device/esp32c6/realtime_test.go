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

func TestBuildFullSpeedExchange(t *testing.T) {
	var requestStates [64]byte
	requestCount := lowspeed.EncodeToken(
		requestStates[:], lowspeed.PIDIn, 7, 1)
	var responseStates [32]byte
	responseCount := lowspeed.EncodeHandshake(responseStates[:], lowspeed.PIDNak)
	request := FullSpeedPacket{States: requestStates[:requestCount]}
	response := FullSpeedPacket{States: responseStates[:responseCount]}
	var exchange FullSpeedExchange
	if !exchange.BuildFullSpeed(&request, &response) {
		t.Fatal("BuildFullSpeed rejected a valid IN/NAK exchange")
	}
	if exchange.Entry() == 0 || exchange.words == 0 {
		t.Fatal("BuildFullSpeed did not publish an exchange entry point")
	}
	foundCycle, stores := false, 0
	for _, instruction := range exchange.code[:exchange.words] {
		if instruction == 0x7e2025f3 {
			foundCycle = true
		}
		if instruction == rvStoreT4AtT0 || instruction == rvStoreT5AtT0 {
			stores++
		}
	}
	if !foundCycle {
		t.Fatal("exchange does not capture the request EOP cycle")
	}
	if want := responseCount - 3 + 1; stores != want {
		t.Fatalf("response differential stores = %d, want %d", stores, want)
	}
	if exchange.code[exchange.words-1] != rvReturn {
		t.Fatal("exchange does not return through ra")
	}
}

func TestBuildFullSpeedReceiveOnlyExchange(t *testing.T) {
	var states [32]byte
	count := lowspeed.EncodeHandshake(states[:], lowspeed.PIDAck)
	request := FullSpeedPacket{States: states[:count]}
	var exchange FullSpeedExchange
	if !exchange.BuildFullSpeed(&request, nil) {
		t.Fatal("BuildFullSpeed rejected a receive-only ACK matcher")
	}
	if exchange.words < 2 || exchange.code[exchange.words-1] != rvReturn {
		t.Fatal("receive-only matcher has no return")
	}
}

func TestBuildFullSpeedExchangeRejectsInvalidPackets(t *testing.T) {
	var exchange FullSpeedExchange
	badRequest := FullSpeedPacket{States: []byte{2, 1, 0, 1, 1}}
	if exchange.BuildFullSpeed(&badRequest, nil) {
		t.Fatal("BuildFullSpeed accepted an invalid request EOP")
	}
	var request [32]byte
	count := lowspeed.EncodeHandshake(request[:], lowspeed.PIDAck)
	validRequest := FullSpeedPacket{States: request[:count]}
	badResponse := FullSpeedPacket{States: []byte{2, 1, 0}}
	if exchange.BuildFullSpeed(&validRequest, &badResponse) {
		t.Fatal("BuildFullSpeed accepted an invalid response EOP")
	}
}

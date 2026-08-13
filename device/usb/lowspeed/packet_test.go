package lowspeed

import "testing"

func TestEncodeData(t *testing.T) {
	var states [128]byte
	count := EncodeData(states[:], 0xc3, []byte{0xff, 0xff, 'P'})
	if count == 0 {
		t.Fatal("EncodeData rejected a valid DATA0 packet")
	}
	wantSync := []byte{LineK, LineJ, LineK, LineJ, LineK, LineJ, LineK, LineK}
	for index, want := range wantSync {
		if states[index] != want {
			t.Fatalf("sync[%d] = %d, want %d", index, states[index], want)
		}
	}
	if states[count-3] != LineSE0 || states[count-2] != LineSE0 || states[count-1] != LineJ {
		t.Fatalf("invalid EOP: %v", states[count-3:count])
	}
	if count <= (1+1+3+2)*8+3 {
		t.Fatalf("packet did not insert a stuffed bit: %d states", count)
	}
	var decoded [8]byte
	pid, length, ok := Decode(decoded[:], states[:count])
	if !ok || pid != PIDData0 || length != 3 || decoded[0] != 0xff || decoded[1] != 0xff || decoded[2] != 'P' {
		t.Fatalf("decoded pid=%02x length=%d data=%v ok=%v", pid, length, decoded[:length], ok)
	}
}

func TestEncodeDataRejectsBadPIDAndShortBuffer(t *testing.T) {
	var short [8]byte
	if EncodeData(short[:], 0xc3, nil) != 0 {
		t.Fatal("short destination accepted")
	}
	var enough [64]byte
	if EncodeData(enough[:], 0x03, nil) != 0 {
		t.Fatal("PID without complement accepted")
	}
}

func TestTokenAndHandshakeRoundTrip(t *testing.T) {
	var states [64]byte
	var data [8]byte
	count := EncodeToken(states[:], PIDSetup, 37, 5)
	pid, length, ok := Decode(data[:], states[:count])
	if !ok || pid != PIDSetup || length != 2 || data[0]&0x7f != 37 || (data[0]>>7|data[1]<<1)&0x0f != 5 {
		t.Fatalf("token pid=%02x length=%d data=%v ok=%v", pid, length, data[:length], ok)
	}
	count = EncodeHandshake(states[:], PIDAck)
	pid, length, ok = Decode(data[:], states[:count])
	if !ok || pid != PIDAck || length != 0 {
		t.Fatalf("handshake pid=%02x length=%d ok=%v", pid, length, ok)
	}
	states[count-2] = LineK
	if _, _, ok = Decode(data[:], states[:count]); ok {
		t.Fatal("malformed EOP accepted")
	}
}

package usbhid

import (
	"bytes"
	"testing"
)

func TestDescriptorConsistency(t *testing.T) {
	configureReports(false)
	if len(device) != int(device[0]) || device[7] != 64 || device[17] != 1 {
		t.Fatal("invalid device descriptor or EP0 size")
	}
	if int(configuration[2])|int(configuration[3])<<8 != len(configuration) {
		t.Fatal("configuration total length disagrees with descriptors")
	}
	interfaces, endpoints := 0, 0
	for at := 0; at < len(configuration); {
		n := int(configuration[at])
		if n < 2 || at+n > len(configuration) {
			t.Fatalf("invalid descriptor at %d", at)
		}
		d := configuration[at : at+n]
		switch d[1] {
		case 4:
			interfaces++
			if n != 9 || d[4] != 1 || d[5] != 3 || d[6] != 0 || d[7] != 0 {
				t.Fatal("expected one non-boot HID interface endpoint")
			}
		case 0x21:
			if n != 9 || d[6] != 0x22 || int(d[7])|int(d[8])<<8 != len(hidReport) {
				t.Fatal("HID report length disagrees with class descriptor")
			}
		case 5:
			endpoints++
			if n != 7 || d[2] != 0x81 || d[3] != 3 || d[4] != 1 || d[5] != 0 || d[6] == 0 {
				t.Fatal("expected one-byte interrupt IN endpoint")
			}
		}
		at += n
	}
	if interfaces != 1 || endpoints != 1 {
		t.Fatal("interface/endpoint count mismatch")
	}
	// Parse HID short items to check that the report occupies one byte,
	// containing two variable button bits and six bits of constant padding.
	size, count, inputBits, buttonBits := 0, 0, 0, 0
	for at := 0; at < len(hidReport); {
		prefix := hidReport[at]
		n := int(prefix & 3)
		if n == 3 {
			n = 4
		}
		if prefix == 0xfe || at+1+n > len(hidReport) {
			t.Fatal("invalid HID item")
		}
		v := 0
		for i := 0; i < n; i++ {
			v |= int(hidReport[at+1+i]) << uint(8*i)
		}
		switch prefix & 0xfc {
		case 0x74:
			size = v
		case 0x94:
			count = v
		case 0x80:
			inputBits += size * count
			if v&1 == 0 {
				buttonBits += size * count
			}
		}
		at += 1 + n
	}
	if inputBits != 8 || buttonBits != 2 {
		t.Fatalf("report contains %d bits, %d variable bits", inputBits, buttonBits)
	}
}

func TestKeyboardPressAndRelease(t *testing.T) {
	previous := identity
	defer func() { identity = previous; report = 0; configureReports(false) }()
	identity.Keyboard = true
	configureReports(true)
	if int(configuration[25]) != len(keyboardDescriptor) || configuration[31] != 8 || keyboardDescriptor[3] != 6 {
		t.Fatal("keyboard descriptor and endpoint disagree")
	}
	for _, tc := range []struct {
		state byte
		want  []byte
	}{
		{1, []byte{0, 0, 4, 0, 0, 0, 0, 0}},
		{3, []byte{0, 0, 4, 5, 0, 0, 0, 0}},
		{2, []byte{0, 0, 5, 0, 0, 0, 0, 0}},
		{0, []byte{0, 0, 0, 0, 0, 0, 0, 0}},
	} {
		report = tc.state
		if got := inputReport(); !bytes.Equal(got, tc.want) {
			t.Fatalf("state %d: got %v, want %v", tc.state, got, tc.want)
		}
	}
}

func TestIdentityFitsSingleControlPacket(t *testing.T) {
	if !validString("Renvo DualKey") || !validString("1234567890123456789012345678901") {
		t.Fatal("valid identity rejected")
	}
	for _, s := range []string{"12345678901234567890123456789012", "bad\x00name", "non-ASCII: é"} {
		if validString(s) {
			t.Fatalf("unsupported identity accepted: %q", s)
		}
	}
}

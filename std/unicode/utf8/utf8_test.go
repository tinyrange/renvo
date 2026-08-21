package utf8

import "testing"

func TestUTF8(t *testing.T) {
	r, size := DecodeRuneInString("a¢€")
	if r != 'a' || size != 1 {
		t.Fatalf("ascii decode failed")
	}
	r, size = DecodeRuneInString("¢€")
	if r != '¢' || size != 2 {
		t.Fatalf("two-byte decode failed: %U %d", r, size)
	}
	if RuneCountInString("a¢€") != 3 || RuneLen('€') != 3 || !ValidString("a¢€") || ValidString(string([]byte{0xff})) {
		t.Fatalf("rune helpers failed")
	}
	buf := make([]byte, 4)
	n := EncodeRune(buf, '€')
	if n != 3 || string(buf[:n]) != "€" {
		t.Fatalf("EncodeRune failed: %d %v", n, buf[:n])
	}
}

func TestByteAPIs(t *testing.T) {
	valid := []byte("Aλ界")
	if !Valid(valid) || RuneCount(valid) != 3 {
		t.Fatal("valid byte APIs failed")
	}
	r, n := DecodeRune(valid[1:])
	if r != 'λ' || n != 2 {
		t.Fatalf("DecodeRune = %U, %d", r, n)
	}
	if Valid([]byte{0xff}) {
		t.Fatal("invalid bytes accepted")
	}
}

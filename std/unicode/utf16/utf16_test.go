package utf16

import "testing"

func TestUTF16(t *testing.T) {
	encoded := Encode([]rune{'A', 0x1f600, -1, 0xd800, 0x110000})
	want := []uint16{65, 0xd83d, 0xde00, 0xfffd, 0xfffd, 0xfffd}
	if len(encoded) != len(want) {
		t.Fatal("encoded length")
	}
	for i := range want {
		if encoded[i] != want[i] {
			t.Fatal("encoded unit", i)
		}
	}
	decoded := Decode([]uint16{0xd83d, 0xde00, 0xd800, 65, 0xdc00})
	if len(decoded) != 4 {
		for _, r := range decoded {
			t.Log("decoded", r)
		}
		t.Fatal("decoded length", len(decoded))
		return
	}
	if decoded[0] != 0x1f600 || decoded[1] != 0xfffd || decoded[2] != 65 || decoded[3] != 0xfffd {
		t.Fatal("decoded code points", len(decoded), decoded[0], decoded[1], decoded[2], decoded[3])
	}
	if string(decoded) != "😀�A�" {
		t.Fatal("decoded runes")
	}
	if DecodeRune(1, 2) != replacement || RuneLen(0xdfff) != -1 {
		t.Fatal("invalid scalar")
	}
}

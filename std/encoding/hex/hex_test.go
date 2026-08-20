package hex

import "testing"

func TestRoundTrip(t *testing.T) {
	s := EncodeToString([]byte("hello"))
	if s != "68656c6c6f" {
		t.Fatal(s)
	}
	b, e := DecodeString(s)
	if e != nil || string(b) != "hello" {
		t.Fatal(string(b), e)
	}
}

package base64

import "testing"

func TestRoundTrip(t *testing.T) {
	cases := []struct {
		e    *Encoding
		want string
	}{{StdEncoding, "aGVsbG8/"}, {RawURLEncoding, "aGVsbG8_"}}
	for _, c := range cases {
		s := c.e.EncodeToString([]byte("hello?"))
		if s != c.want {
			t.Fatal(s)
		}
		b, e := c.e.DecodeString(s)
		if e != nil || string(b) != "hello?" {
			t.Fatal(string(b), e)
		}
	}
}

func TestMalformedInput(t *testing.T) {
	for _, input := range []string{"!", "a", "ab$c", "ab=c"} {
		if _, err := RawURLEncoding.DecodeString(input); err == nil {
			t.Fatalf("DecodeString(%q) unexpectedly succeeded", input)
		}
	}
}

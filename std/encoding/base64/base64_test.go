package base64

import "testing"

func TestKnownVectors(t *testing.T) {
	inputs := []string{"", "f", "fo", "foo", "foob", "fooba", "foobar", "\x00\xff\x80"}
	encoded := []string{"", "Zg==", "Zm8=", "Zm9v", "Zm9vYg==", "Zm9vYmE=", "Zm9vYmFy", "AP+A"}
	for i := range inputs {
		if got := StdEncoding.EncodeToString([]byte(inputs[i])); got != encoded[i] {
			t.Fatal("encoding", i, got)
		}
		got, err := StdEncoding.DecodeString(encoded[i])
		if err != nil || string(got) != inputs[i] {
			t.Fatal("decoding", i, err)
		}
		raw := RawStdEncoding.EncodeToString([]byte(inputs[i]))
		got, err = RawStdEncoding.DecodeString(raw)
		if err != nil || string(got) != inputs[i] {
			t.Fatal("raw roundtrip", i)
		}
	}
	if RawURLEncoding.EncodeToString([]byte{255, 255}) != "__8" {
		t.Fatal("URL alphabet")
	}
	got, err := StdEncoding.DecodeString("Z\r\ng=\n=\r\n")
	if err != nil || string(got) != "f" {
		t.Fatal("newlines", err)
	}
}

func TestMalformedAndStrict(t *testing.T) {
	for _, s := range []string{"Z", "Zg", "=g==", "Zg=", "Zg==x", "Z g==", "!!!!", "Zm9v="} {
		if _, err := StdEncoding.DecodeString(s); err == nil {
			t.Fatal("accepted malformed", s)
		}
	}
	for _, s := range []string{"Z", "Zg=", "Z g", "!!!!"} {
		if _, err := RawStdEncoding.DecodeString(s); err == nil {
			t.Fatal("accepted malformed raw", s)
		}
	}
	if _, err := StdEncoding.Strict().DecodeString("Zh=="); err == nil {
		t.Fatal("nonzero padding bits")
	}
	if _, err := RawStdEncoding.Strict().DecodeString("Zm9"); err == nil {
		t.Fatal("nonzero raw padding bits")
	}
	if got, err := StdEncoding.DecodeString("Zh=="); err != nil || string(got) != "f" {
		t.Fatal("non-strict decode")
	}
}

func TestAllByteValuesAndAppend(t *testing.T) {
	for n := 0; n < 260; n++ {
		src := make([]byte, n)
		for i := range src {
			src[i] = byte(i)
		}
		encoded := RawStdEncoding.AppendEncode([]byte("prefix:"), src)
		decoded, err := RawStdEncoding.AppendDecode([]byte("x"), encoded[7:])
		if err != nil || string(decoded[1:]) != string(src) || decoded[0] != 'x' {
			t.Fatal("roundtrip", n, err)
		}
	}
}

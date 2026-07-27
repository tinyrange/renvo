package rtgb

import (
	"bytes"
	"testing"

	"renvo.dev/internal/rtg"
)

func TestArtifactRoundTrip(t *testing.T) {
	var definition [32]byte
	definition[0] = 7
	want := Artifact{
		Descriptor: rtg.TargetDescriptor{
			Name: "test/tiny64", OS: "test", ISA: "tiny64",
			WordBits: 64, PointerBits: 64, Endian: "little",
			ABI: "tiny", Runtime: "tiny", Executable: "tiny",
			Aliases: []string{"tiny"}, BuildTags: []string{"tiny64"},
			Capabilities: []string{"executable"}, Definition: definition, Version: 1,
		},
		Host: "linux/amd64", Generator: 1, Kernel: 1, Payload: []byte("payload"),
	}
	encoded, ok := Encode(want)
	if !ok {
		t.Fatal("Encode failed")
	}
	got, ok := Decode(encoded)
	if !ok {
		t.Fatal("Decode failed")
	}
	if got.Descriptor.Name != want.Descriptor.Name || got.Host != want.Host ||
		got.Descriptor.Definition != want.Descriptor.Definition ||
		!bytes.Equal(got.Payload, want.Payload) {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}
}

func TestArtifactRejectsCorruptionAndTruncation(t *testing.T) {
	artifact := Artifact{
		Descriptor: rtg.TargetDescriptor{
			Name: "test/tiny", OS: "test", ISA: "tiny",
			WordBits: 32, PointerBits: 32, Endian: "little",
			ABI: "tiny", Runtime: "tiny", Executable: "tiny", Version: 1,
		},
		Host: "linux/amd64", Generator: 1, Kernel: 1, Payload: []byte("payload"),
	}
	encoded, _ := Encode(artifact)
	for i := 0; i < len(encoded); i++ {
		if _, ok := Decode(encoded[:i]); ok {
			t.Fatalf("accepted truncation at %d", i)
		}
	}
	corrupt := append([]byte(nil), encoded...)
	corrupt[len(corrupt)-1] ^= 1
	if _, ok := Decode(corrupt); ok {
		t.Fatal("accepted corrupt artifact")
	}
}

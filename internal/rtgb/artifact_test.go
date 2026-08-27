package rtgb

import (
	"bytes"
	"reflect"
	"testing"

	"renvo.dev/internal/rbe"
	"renvo.dev/internal/rtg"
)

func TestArtifactRoundTrip(t *testing.T) {
	var definition [32]byte
	definition[0] = 7
	want := Artifact{
		Descriptor: rtg.TargetDescriptor{
			Name: "test/tiny64", Family: rtg.BackendFamilyNativeV1, OS: "test", ISA: "tiny64",
			WordBits: 64, PointerBits: 64, CodePointerBits: 64,
			FunctionPointerBits: 64, MaxAlign: 8, Endian: "little",
			ABI: "tiny", Runtime: "tiny", OutputKind: "tiny", Executable: "tiny",
			Aliases: []string{"tiny"}, BuildTags: []string{"tiny64"},
			Capabilities: []string{"executable"}, RuntimeOps: []string{},
			Definition: definition, Version: 1,
		},
		Host: "linux/amd64", Generator: 1, Kernel: 1, Protocol: 1,
		Unit: 1, Optimization: 1, Payload: []byte("payload"),
		LibraryFiles: []rbe.File{{Path: "syscall/v7.go", Source: []byte("package syscall\n")}},
	}
	want.Enablement[0] = 9
	encoded, ok := Encode(want)
	if !ok {
		t.Fatal("Encode failed")
	}
	got, ok := Decode(encoded)
	if !ok {
		t.Fatal("Decode failed")
	}
	if !reflect.DeepEqual(got.Descriptor, want.Descriptor) ||
		got.Host != want.Host || got.Generator != want.Generator ||
		got.Kernel != want.Kernel || got.Protocol != want.Protocol ||
		got.Unit != want.Unit || got.Optimization != want.Optimization ||
		got.Enablement != want.Enablement || !reflect.DeepEqual(got.LibraryFiles, want.LibraryFiles) ||
		!bytes.Equal(got.Payload, want.Payload) {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}
	want.LibraryFiles[0].Path = "../escape.go"
	if _, ok := Encode(want); ok {
		t.Fatal("Encode accepted an unsafe library overlay path")
	}
}

func TestArtifactRejectsCorruptionAndTruncation(t *testing.T) {
	artifact := Artifact{
		Descriptor: rtg.TargetDescriptor{
			Name: "test/tiny", Family: rtg.BackendFamilyNativeV1, OS: "test", ISA: "tiny",
			WordBits: 32, PointerBits: 32, CodePointerBits: 32,
			FunctionPointerBits: 32, MaxAlign: 4, Endian: "little",
			ABI: "tiny", Runtime: "tiny", OutputKind: "tiny", Executable: "tiny", Version: 1,
		},
		Host: "linux/amd64", Generator: 1, Kernel: 1, Protocol: 1,
		Unit: 1, Optimization: 1, Payload: []byte("payload"),
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

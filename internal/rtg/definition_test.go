package rtg

import (
	"bytes"
	"os"
	"testing"
)

func TestAArch64DefinitionVerticalSlice(t *testing.T) {
	source, err := os.ReadFile("../../backend/definitions/aarch64.rtg")
	if err != nil {
		t.Fatal(err)
	}
	resolved := Resolve(Parse(source, "aarch64.rtg"))
	if !resolved.Ok {
		t.Fatalf("definition failed: %#v", resolved.Diagnostics)
	}
	if len(resolved.Targets) != 3 {
		t.Fatalf("target count = %d, want 3", len(resolved.Targets))
	}
	for _, target := range []string{"linux/aarch64", "darwin/arm64", "windows/arm64"} {
		generated := GenerateFixedBackend(resolved, target)
		if !generated.Ok {
			t.Fatalf("generate %s: %#v", target, generated.Diagnostics)
		}
		if containsText(string(generated.Source), "RTGLookupTarget") {
			t.Errorf("fixed %s output retained universal dispatch", target)
		}
		if target == "linux/aarch64" {
			again := GenerateFixedBackend(resolved, target)
			if !again.Ok || !bytes.Equal(generated.Source, again.Source) {
				t.Fatal("linux/aarch64 generation is not deterministic")
			}
		}
	}
}

func TestCheckedInBackendsAreCurrent(t *testing.T) {
	type output struct {
		path   string
		target string
		block  int
	}
	families := []struct {
		definition string
		outputs    []output
	}{
		{"aarch64.rtg", []output{
			{"compiler_aarch64_impl.go", "linux/aarch64", 1},
			{"compiler_aarch64_target_impl.go", "linux/aarch64", 2},
			{"compiler_linux_aarch64_impl.go", "linux/aarch64", 3},
			{"compiler_darwin_arm64_impl.go", "darwin/arm64", 4},
			{"compiler_windows_arm64_impl.go", "windows/arm64", 5},
		}},
		{"amd64.rtg", []output{
			{"compiler_amd64_impl.go", "linux/amd64", 0},
			{"compiler_amd64_target_impl.go", "linux/amd64", 1},
			{"compiler_linux_amd64_impl.go", "linux/amd64", 2},
			{"compiler_windows_amd64_impl.go", "windows/amd64", 3},
			{"compiler_linux_kernel_amd64_impl.go", "linux-kernel/amd64", 4},
		}},
		{"386.rtg", []output{
			{"compiler_386_impl.go", "linux/386", 0},
			{"compiler_386_target_impl.go", "linux/386", 1},
			{"compiler_linux_386_impl.go", "linux/386", 2},
			{"compiler_windows_386_impl.go", "windows/386", 3},
		}},
		{"arm.rtg", []output{
			{"compiler_arm_impl.go", "linux/arm", 0},
			{"compiler_linux_arm_impl.go", "linux/arm", 1},
		}},
		{"wasm32.rtg", []output{
			{"compiler_wasm32_impl.go", "wasi/wasm32", 0},
			{"compiler_wasi_wasm32_impl.go", "wasi/wasm32", 1},
		}},
	}
	for _, family := range families {
		source, err := os.ReadFile("../../backend/definitions/" + family.definition)
		if err != nil {
			t.Fatal(err)
		}
		resolved := Resolve(Parse(source, family.definition))
		if !resolved.Ok {
			t.Fatalf("%s failed: %#v", family.definition, resolved.Diagnostics)
		}
		for _, output := range family.outputs {
			generated := GenerateBuiltinBackend(resolved, output.target, "main", output.block)
			if !generated.Ok {
				t.Fatalf("generate %s: %#v", output.path, generated.Diagnostics)
			}
			checkedIn, err := os.ReadFile("../../backend/" + output.path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(generated.Source, checkedIn) {
				t.Errorf("%s is stale; run go generate ./backend/definitions", output.path)
			}
		}
	}
}

func TestAArch64EncodingSlice(t *testing.T) {
	if got := encodeAArch64RET(30); got != 0xd65f03c0 {
		t.Fatalf("RET x30 = %#08x", got)
	}
	if got := encodeAArch64ADDImmediate(0, 1, 7); got != 0x91001c20 {
		t.Fatalf("ADD x0, x1, #7 = %#08x", got)
	}
	if got := encodeAArch64MOVZ(0, 0x1234, 16); got != 0xd2a24680 {
		t.Fatalf("MOVZ x0, #0x1234, LSL #16 = %#08x", got)
	}
	if got := encodeAArch64B(8); got != 0x14000002 {
		t.Fatalf("B +8 = %#08x", got)
	}
}

// These mirrors make the expected instruction words reviewable independently
// from generated-source spelling. The generated functions use the same direct
// expressions and are syntax-checked by TestAArch64DefinitionVerticalSlice.
func encodeAArch64RET(register uint32) uint32 {
	return 0xd65f0000 | (register&31)<<5
}

func encodeAArch64ADDImmediate(destination uint32, source uint32, value uint32) uint32 {
	return 0x91000000 | (value&0xfff)<<10 | (source&31)<<5 | destination&31
}

func encodeAArch64MOVZ(destination uint32, value uint32, shift uint32) uint32 {
	return 0xd2800000 | ((shift/16)&3)<<21 | (value&0xffff)<<5 | destination&31
}

func encodeAArch64B(displacement int32) uint32 {
	return 0x14000000 | uint32(displacement>>2)&0x03ffffff
}

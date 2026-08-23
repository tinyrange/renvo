package main

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"renvo.dev/std/vm"
)

func TestIEEEFloatLiteralBits(t *testing.T) {
	for _, literal := range []string{
		"0.02",
		"2.0",
		"0x1p-1074",
		"0x1.00000000000008p0",
		"0x1.00000000000018p0",
		"0.560975609756",
		"0.560975609757",
	} {
		program := renvoParseProgram([]byte("package main\nvar value float64 = " + literal + "\n"))
		if !program.ok {
			t.Fatalf("parse %q failed", literal)
		}
		token := -1
		for i := 0; i < renvoTokCount(&program); i++ {
			if renvoTokIsKind(&program, i, renvoTokFloat) {
				token = i
				break
			}
		}
		if token < 0 {
			t.Fatalf("float token %q not found", literal)
		}
		value, err := strconv.ParseFloat(literal, 64)
		if err != nil {
			t.Fatalf("host parse %q: %v", literal, err)
		}
		got := renvoParseFloatTokenBits(&program, token, 52, 11, 1023)
		want := math.Float64bits(value)
		if got != want {
			t.Fatalf("literal %q bits = %#016x, want %#016x", literal, got, want)
		}
	}
}

func TestIEEEFloatVM32Corpus(t *testing.T) {
	paths, err := filepath.Glob("tests/ieee_*.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no IEEE regressions")
	}
	for _, path := range paths {
		path := path
		t.Run(strings.TrimSuffix(filepath.Base(path), ".go"), func(t *testing.T) {
			resetRuntime()
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			image, ok := RenvoCompileSourceToBytesWithOptions(source, "vm/vm32", RenvoCompileOptions{
				ArenaSize:    8 * 1024 * 1024,
				StripSymbols: true,
			})
			if !ok {
				t.Fatal("compile vm/vm32")
			}
			result := vm.RunConfig(image, vm.Config{
				Limits: vm.Limits{Steps: 500 * 1000 * 1000, Memory: 16 * 1024 * 1024},
				Args:   []string{"program"},
				Env:    []string{"PATH=/vm"},
			})
			if result.Trap != vm.TrapNone || result.ExitCode != 0 || string(result.Output)+string(result.Stderr) != "PASS\n" {
				t.Fatalf("exit %d, trap %d at pc %d opcode %d, stdout %q, stderr %q, steps %d",
					result.ExitCode, result.Trap, result.TrapPC, result.TrapOpcode, result.Output, result.Stderr, result.Steps)
			}
		})
	}
}

func TestIEEEFloatWASINativeCorpusCompiles(t *testing.T) {
	paths, err := filepath.Glob("tests/ieee_*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		image, ok := RenvoCompileSourceToBytesWithOptions(source, "wasi/wasm32", RenvoCompileOptions{
			ArenaSize:    8 * 1024 * 1024,
			StripSymbols: true,
		})
		if !ok {
			t.Fatalf("compile %s for wasi/wasm32", path)
		}
		if len(image) < 8 || string(image[:4]) != "\x00asm" {
			t.Fatalf("%s produced an invalid WebAssembly header", path)
		}
	}
}

func TestSoftFloatSourceDetection(t *testing.T) {
	source, err := os.ReadFile("compiler_common_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	compiler := renvoParseProgram(source)
	if !compiler.ok || renvoProgramNeedsSoftFloat(&compiler) {
		t.Fatal("compiler implementation unexpectedly requires runtime soft-float helpers")
	}
	program := renvoParseProgram([]byte("package main\nvar value float64 = 0.02\n"))
	if !program.ok || !renvoProgramNeedsSoftFloat(&program) {
		t.Fatal("float source did not request runtime soft-float helpers")
	}
}

func FuzzIEEEFloatLiteralBits(f *testing.F) {
	for _, literal := range []string{
		"0.02", "1e-45", "3.4028235e38", "5e-324", "1.7976931348623157e308",
		"0x1p-149", "0x1.fffffep127", "0x1p-1074", "0x1.fffffffffffffp1023",
	} {
		f.Add(literal)
	}
	f.Fuzz(func(t *testing.T, literal string) {
		if len(literal) == 0 || len(literal) > 128 || literal[0] == '+' || literal[0] == '-' {
			return
		}
		program := renvoParseProgram([]byte("package main\nvar value = " + literal + "\n"))
		if !program.ok {
			return
		}
		token := -1
		for i := 0; i < renvoTokCount(&program); i++ {
			if renvoTokIsKind(&program, i, renvoTokFloat) {
				token = i
				break
			}
		}
		if token < 0 {
			return
		}
		for _, bitSize := range []int{32, 64} {
			value, err := strconv.ParseFloat(literal, bitSize)
			if err != nil && value == 0 {
				return
			}
			fractionBits, exponentBits, bias := 52, 11, 1023
			want := math.Float64bits(value)
			if bitSize == 32 {
				fractionBits, exponentBits, bias = 23, 8, 127
				want = uint64(math.Float32bits(float32(value)))
			}
			got := renvoParseFloatTokenBits(&program, token, fractionBits, exponentBits, bias)
			if got != want {
				t.Fatalf("literal %q/%d bits = %#016x, want %#016x", literal, bitSize, got, want)
			}
		}
	})
}

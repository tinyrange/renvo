package main

import (
	"fmt"
	"testing"

	"renvo.dev/std/vm"
)

func TestVM32UncaughtIndexBounds(t *testing.T) {
	for _, value := range []struct {
		name, source string
		first        int
	}{
		{"bytes", "[]byte{11, 12, 13}", 11},
		{"words", "[]int{11, 12, 13}", 11},
		{"string", "\"abc\"", 97},
	} {
		for _, index := range []int{-2147483648, -1, 0, 2, 3, 2147483647} {
			t.Run(fmt.Sprintf("%s/%d", value.name, index), func(t *testing.T) {
				source := fmt.Sprintf("package main\nfunc read(index int) int { value := %s; return int(value[index]) }\nfunc appMain() int { if read(%d) != %d { return 1 }; print(\"PASS\\n\"); return 0 }\n", value.source, index, value.first+index)
				image, ok := RenvoCompileSourceToBytesWithOptions([]byte(source), "vm/vm32", RenvoCompileOptions{ArenaSize: 8192, StripSymbols: true})
				if !ok {
					t.Fatal("compile VM index program")
				}
				result := vm.Run(image, vm.Limits{Steps: 100000, Memory: 32768})
				if result.Trap != vm.TrapNone {
					t.Fatalf("VM trapped: %d", result.Trap)
				}
				if index >= 0 && index < 3 {
					if result.ExitCode != 0 || string(result.Output) != "PASS\n" || len(result.Stderr) != 0 {
						t.Fatalf("valid index: exit=%d stdout=%q stderr=%q", result.ExitCode, result.Output, result.Stderr)
					}
				} else if result.ExitCode != 2 || len(result.Output) != 0 || string(result.Stderr) != "panic\n" {
					t.Fatalf("invalid index: exit=%d stdout=%q stderr=%q", result.ExitCode, result.Output, result.Stderr)
				}
			})
		}
	}
}

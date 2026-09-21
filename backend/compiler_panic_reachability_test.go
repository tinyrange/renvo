package main

import (
	"bytes"
	"testing"
)

func TestUnreachablePanicDoesNotExpandExecutable(t *testing.T) {
	const program = "package main\nfunc appMain() int { print(\"PASS\\n\"); return 0 }\n"
	for _, target := range []string{"linux/amd64", "linux/386", "linux/arm", "linux/aarch64", "wasi/wasm32", "vm/vm32"} {
		t.Run(target, func(t *testing.T) {
			plain, ok := RenvoCompileSourceToBytesStrip([]byte(program), target, true)
			if !ok {
				t.Fatal("compile baseline")
			}
			extra, ok := RenvoCompileSourceToBytesStrip([]byte(program+"func unused() { panic(\"unreachable\") }\n"), target, true)
			if !ok {
				t.Fatal("compile unused panic")
			}
			if !bytes.Equal(plain, extra) {
				t.Fatalf("unused panic changed executable: %d -> %d bytes", len(plain), len(extra))
			}
		})
	}
}

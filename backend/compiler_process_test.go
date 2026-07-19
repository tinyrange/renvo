package main

import "testing"

func TestRuntimeExitIntrinsicCompilesForEveryTarget(t *testing.T) {
	source := []byte(`package main

func renvo_runtime_Exit(code int) {}

func appMain() int {
	renvo_runtime_Exit(23)
	return 0
}
`)
	for _, target := range []string{
		"linux/amd64",
		"linux/386",
		"linux/aarch64",
		"linux/arm",
		"windows/amd64",
		"windows/386",
		"windows/arm64",
		"wasi/wasm32",
		"darwin/arm64",
	} {
		if _, ok := RenvoCompileSourceToBytes(source, target); !ok {
			t.Errorf("runtime exit intrinsic did not compile for %s", target)
		}
	}
}

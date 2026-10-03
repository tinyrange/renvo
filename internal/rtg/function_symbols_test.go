//go:build !renvo

package rtg

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestPreparedFunctionSymbolsAreDeclaredNotNamed(t *testing.T) {
	const filename = "../../backend/definitions/wasm32.rtg"
	source, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		kind     string
		declared bool
	}{
		{"wasm", false}, {"html-wasm", false}, {"rnvm", false}, {"private-image", true},
	} {
		kind, declared := tc.kind, tc.declared
		modified := bytes.ReplaceAll(source, []byte("target wasi/wasm32 {"), []byte("target custom/image {"))
		modified = bytes.Replace(modified, []byte("kind = wasm"), []byte("kind = \""+kind+"\""), 1)
		if !declared {
			modified = bytes.ReplaceAll(modified, []byte(", function_symbols"), nil)
		}
		resolved := Resolve(Parse(modified, filename))
		generated := GeneratePreparedBackend(resolved, "custom/image")
		if !generated.Ok {
			t.Fatalf("kind %s declared %v: %#v", kind, declared, generated.Diagnostics)
		}
		want := "0"
		if declared {
			want = "1"
		}
		if !strings.Contains(string(generated.Source), "const renvoRTGPreparedFunctionSymbols = "+want) {
			t.Fatalf("kind %s declared %v: incorrect function-symbol policy", kind, declared)
		}
	}
}

//go:build !renvo

package rtg

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestPreparedStaticCallPolicyIsDefinitionBound(t *testing.T) {
	const filename = "../../backend/definitions/darwin_aarch64.rtg"
	source, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	const original = "static_call_layout = split_register_words8"
	for _, tc := range []struct {
		name, policy, want              string
		noEmitter, narrow, kernel, fail bool
	}{
		{name: "renamed", policy: original, want: "renvoStaticCallSplitRegisters"},
		{name: "plain-words", want: "renvoStaticCallWords"},
		{name: "unavailable", noEmitter: true, want: "renvoStaticCallUnavailable"},
		{name: "missing-emitter", policy: original, noEmitter: true, fail: true},
		{name: "narrow", policy: original, narrow: true, fail: true},
		{name: "kernel", policy: original, kernel: true, fail: true},
		{name: "unknown", policy: "static_call_layout = magic", fail: true},
		{name: "duplicate", policy: original + "\n" + original, fail: true},
		{name: "block", policy: "static_call_layout { layout = split_register_words8 }", fail: true},
		{name: "extra", policy: original + " extra", fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			modified := bytes.Replace(source, []byte(original), []byte(tc.policy), 1)
			modified = bytes.ReplaceAll(modified, []byte("darwin_arm64"), []byte("private_runtime"))
			modified = bytes.Replace(modified, []byte("target darwin/arm64 {"), []byte("target private/calls {"), 1)
			modified = bytes.Replace(modified, []byte("os = darwin"), []byte("os = private_environment"), 1)
			if tc.noEmitter {
				modified = bytes.Replace(modified, []byte("emit_static_call = go machStaticCall"), nil, 1)
			}
			resolved := Resolve(ParseImports(modified, filename, testFilesystemImportLoader{}))
			if !resolved.Ok && !tc.fail {
				t.Fatalf("resolve: %#v", resolved.Diagnostics)
			}
			for i := range resolved.Targets {
				if tc.narrow {
					resolved.Targets[i].Descriptor.WordBits = 32
				}
				if tc.kernel {
					resolved.Targets[i].Descriptor.Capabilities = append(resolved.Targets[i].Descriptor.Capabilities, "kernel_module")
				}
			}
			generated := GeneratePreparedBackend(resolved, "private/calls")
			if tc.fail {
				if generated.Ok || len(generated.Source) != 0 {
					t.Fatal("invalid static-call policy generated code")
				}
				for _, diagnostic := range generated.Diagnostics {
					if diagnostic.Code == "RTG-VALIDATE-133" || tc.name == "duplicate" && diagnostic.Code == "RTG-VALIDATE-060" {
						return
					}
				}
				t.Fatalf("missing policy diagnostic: %#v", generated.Diagnostics)
			}
			if !generated.Ok {
				t.Fatalf("generate: %#v", generated.Diagnostics)
			}
			if !strings.Contains(string(generated.Source), "const renvoRTGStaticCallPolicy = "+tc.want) {
				t.Fatal("policy did not follow the declaration")
			}
		})
	}
}

//go:build !renvo

package rtg

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

type objectPolicyImportLoader struct {
	policy string
	rename bool
}

func (loader objectPolicyImportLoader) rewrite(source []byte) []byte {
	source = bytes.ReplaceAll(source, []byte("object_call_layout = sysv_eightbyte"), []byte(loader.policy))
	if loader.rename {
		source = bytes.ReplaceAll(source, []byte("sysv_x86_64"), []byte("private_call_abi"))
	}
	return source
}

func (loader objectPolicyImportLoader) LoadImport(filename, path string) ImportSource {
	source := (testFilesystemImportLoader{}).LoadImport(filename, path)
	source.Source = loader.rewrite(source.Source)
	return source
}

func TestPreparedObjectCallPolicyIsDefinitionBound(t *testing.T) {
	const filename = "../../backend/definitions/linux_amd64.rtg"
	source, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	const original = "object_call_layout = sysv_eightbyte"
	for _, tc := range []struct {
		name, policy                    string
		rename, noEmitter, narrow, fail bool
		wantLayout, wantCall            bool
	}{
		{name: "renamed", policy: original, rename: true, wantLayout: true, wantCall: true},
		{name: "absent-familiar-name"},
		{name: "no-runtime-emitter", policy: original, noEmitter: true, wantLayout: true},
		{name: "narrow-words", policy: original, narrow: true, fail: true},
		{name: "unknown", policy: "object_call_layout = magic", fail: true},
		{name: "duplicate", policy: original + "\n" + original, fail: true},
		{name: "block", policy: "object_call_layout { layout = sysv_eightbyte }", fail: true},
		{name: "extra-value", policy: original + " extra", fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			loader := objectPolicyImportLoader{policy: tc.policy, rename: tc.rename}
			modified := loader.rewrite(source)
			modified = bytes.Replace(modified, []byte("target linux/amd64 {"), []byte("target private/object {"), 1)
			modified = bytes.Replace(modified, []byte("os = linux"), []byte("os = private_environment"), 1)
			if tc.noEmitter {
				modified = bytes.Replace(modified, []byte("emit_static_call = go linuxObjectStaticCall"), nil, 1)
			}
			resolved := Resolve(ParseImports(modified, filename, loader))
			if !resolved.Ok && !tc.fail {
				t.Fatalf("resolve fixture: %#v", resolved.Diagnostics)
			}
			if tc.narrow {
				for i := range resolved.Targets {
					resolved.Targets[i].Descriptor.WordBits = 32
				}
			}
			generated := GeneratePreparedBackend(resolved, "private/object")
			if tc.fail {
				if generated.Ok || len(generated.Source) != 0 {
					t.Fatal("invalid object-call layout generated output")
				}
				for _, diagnostic := range generated.Diagnostics {
					if diagnostic.Code == "RTG-VALIDATE-132" || tc.name == "duplicate" && diagnostic.Code == "RTG-VALIDATE-060" {
						return
					}
				}
				t.Fatalf("missing layout diagnostic: %#v", generated.Diagnostics)
			}
			if !generated.Ok {
				t.Fatalf("generate fixture: %#v", generated.Diagnostics)
			}
			layout, call := "0", "renvoObjectABIUnavailable"
			if tc.wantLayout {
				layout = "16"
			}
			if tc.wantCall {
				call = "renvoObjectABISysV"
			}
			for _, want := range []string{
				"const renvoRTGObjectAggregateRegisterBytes = " + layout,
				"const renvoRTGObjectCallABI = " + call,
				"return renvoRTGObjectCallABI",
				"return renvoRTGObjectAggregateRegisterBytes",
			} {
				if !strings.Contains(string(generated.Source), want) {
					t.Errorf("missing %q", want)
				}
			}
		})
	}
}

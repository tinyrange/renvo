//go:build renvo_bundle

package frontend_tests

import (
	"bytes"
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"renvo.dev/driver"
	"runtime"
	"testing"
	"testing/fstest"
)

func TestFrontendCConstantAndInlineMemory(t *testing.T) {
	for _, fixture := range []string{"c_sizeof_unary_constant", "c_c11_inline_linkage"} {
		t.Run(fixture, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join(repoRoot(t), "backend", "tests", fixture+".c"))
			if err != nil {
				t.Fatal(err)
			}
			fs := linkSourceFS{fstest.MapFS{"main.c": {Data: source}}}
			obj, err := driver.CompileCommand(&driver.CommandRequest{Filesystem: fs, Args: []string{"cc", "-c", "main.c"}, Target: "linux/amd64"})
			if err != nil || !obj.Ok {
				t.Fatalf("object %v %+v", err, obj)
			}
			if fixture == "c_c11_inline_linkage" {
				f, err := elf.NewFile(bytes.NewReader(obj.Binary))
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				symbols, err := f.Symbols()
				if err != nil {
					t.Fatal(err)
				}
				bindings := map[string]elf.SymBind{}
				for _, s := range symbols {
					if s.Section != elf.SHN_UNDEF {
						bindings[s.Name] = elf.ST_BIND(s.Info)
					}
				}
				for _, n := range []string{"exported_add", "later_export"} {
					if b, ok := bindings[n]; !ok || b != elf.STB_GLOBAL {
						t.Fatalf("%s binding %v present %v", n, b, ok)
					}
				}
				for _, n := range []string{"header_add", "gnu_local"} {
					if b, ok := bindings[n]; ok && b == elf.STB_GLOBAL {
						t.Fatalf("%s globally exported", n)
					}
				}
			}
			fs.MapFS["main.o"] = &fstest.MapFile{Data: obj.Binary}
			linked, err := driver.CompileCommand(&driver.CommandRequest{Filesystem: fs, Args: []string{"cc", "main.o"}, Target: "linux/amd64"})
			if err != nil || !linked.Ok {
				t.Fatalf("link %v %+v", err, linked)
			}
			if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
				name := filepath.Join(t.TempDir(), "app")
				if err = os.WriteFile(name, linked.Binary, 0700); err != nil {
					t.Fatal(err)
				}
				if out, err := exec.Command(name).CombinedOutput(); err != nil || string(out) != "PASS\n" {
					t.Fatalf("%v %q", err, out)
				}
			}
		})
	}
}

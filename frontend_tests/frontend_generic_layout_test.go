package frontend_tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"renvo.dev/internal/targetinfo"
)

func TestFrontendGenericLayoutObjectSystemLink(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("Go object execution requires Linux/amd64")
	}
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("system C linker unavailable")
	}
	root := repoRoot(t)
	for _, frontend := range []struct {
		name   string
		config frontendConfig
	}{{"host", frontendCompiler(t, root)}, {"stage3", selfHostedFrontendCompiler(t, root)}} {
		t.Run(frontend.name, func(t *testing.T) {
			dir := t.TempDir()
			files := map[string]string{
				"go.mod": "module example.com/layout\ngo 1.25\n",
				"bridge.go": `package bridge
import "unsafe"
func layout[T any]()int64 {
 var record struct{ A byte; B byte; C *T }
 const size=unsafe.Sizeof(record)
 const offset=unsafe.Offsetof(record.C)
 var bytes [size]byte
 return int64(len(bytes)*100+int(offset))
}
//export renvo_generic_layout
func Layout()int64{return layout[int]()}
`,
				"harness.c": "#include <stdint.h>\nextern int64_t renvo_generic_layout(void);\nint main(void){return renvo_generic_layout()==1608 ? 0 : 1;}\n",
			}
			for name, source := range files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0644); err != nil {
					t.Fatal(err)
				}
			}
			object, program := filepath.Join(dir, "bridge.o"), filepath.Join(dir, "program")
			cmd := frontendCommand(frontend.config, "-mode=object", "-o", object, "bridge.go")
			cmd.Dir, cmd.Env = dir, frontendCommandEnv(frontend.config.env, dir)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("compile generic object: %v\n%s", err, output)
			}
			if output, err := exec.Command(cc, filepath.Join(dir, "harness.c"), object, "-o", program).CombinedOutput(); err != nil {
				t.Fatalf("link generic object: %v\n%s", err, output)
			}
			if output, err := exec.Command(program).CombinedOutput(); err != nil {
				t.Fatalf("run generic object: %v\n%s", err, output)
			}
		})
	}
}

func TestFrontendGenericLayoutArrayIdentityByTarget(t *testing.T) {
	root := repoRoot(t)
	for _, frontend := range []struct {
		name   string
		config frontendConfig
	}{{"host", frontendCompiler(t, root)}, {"stage3", selfHostedFrontendCompiler(t, root)}} {
		t.Run(frontend.name, func(t *testing.T) {
			for _, target := range targetinfo.All() {
				if !target.Advertised || target.Virtual {
					continue
				}
				t.Run(target.Name, func(t *testing.T) {
					align, size := 4, 12
					if target.WordBits == 64 || target.ISA == "wasm32" {
						align, size = 8, 16
					}
					for _, valid := range []bool{true, false} {
						pointer := 8
						if !valid {
							pointer++
						}
						dir := t.TempDir()
						if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/layout\ngo 1.25\n"), 0644); err != nil {
							t.Fatal(err)
						}
						source := fmt.Sprintf(`package main
import "unsafe"
func Pointer[T any]() [unsafe.Sizeof((*T)(nil))]T {var v [unsafe.Sizeof((*T)(nil))]T;return v}
func Aggregate[T any]() [unsafe.Sizeof(struct{A byte;B uint64}{})]T {var v [unsafe.Sizeof(struct{A byte;B uint64}{})]T;return v}
func Alignment[T any]() [unsafe.Alignof(uint64(0))]T {var v [unsafe.Alignof(uint64(0))]T;return v}
func main(){var p [%d]int=Pointer[int]();var a [%d]int=Aggregate[int]();var b [%d]int=Alignment[int]();_=p;_=a;_=b}
`, pointer, size, align)
						if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(source), 0644); err != nil {
							t.Fatal(err)
						}
						cmd := frontendCommand(frontend.config, "-t", target.Name, "-emit-unit", "-o", filepath.Join(dir, "program.unit"), ".")
						cmd.Dir = dir
						cmd.Env = frontendCommandEnv(frontend.config.env, dir)
						output, err := cmd.CombinedOutput()
						if valid && err != nil {
							t.Fatalf("compile target layout identity: %v\n%s", err, output)
						}
						if !valid && (err == nil || !strings.Contains(string(output), "RENVO-CHECK-042")) {
							t.Fatalf("expected array identity rejection: %v\n%s", err, output)
						}
					}
				})
			}
		})
	}
}

package frontend_tests

import (
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFrontendGoObjectRoots(t *testing.T) {
	root := repoRoot(t)
	for _, compiler := range []struct {
		name   string
		config frontendConfig
	}{
		{"host", frontendCompiler(t, root)},
		{"stage3", selfHostedFrontendCompiler(t, root)},
	} {
		t.Run(compiler.name, func(t *testing.T) {
			for _, mode := range []string{"-c", "-mode=object"} {
				t.Run(mode, func(t *testing.T) {
					dir := t.TempDir()
					write := func(name, text string) {
						t.Helper()
						if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0644); err != nil {
							t.Fatal(err)
						}
					}
					write("go.mod", "module example.com/objectprobe\ngo 1.23\n")
					write("main.go", "package main\nfunc Add(a,b int)int{return a+b}\nfunc main(){if Add(19,23)!=42{panic(\"add\")}}\n")
					run := func(args ...string) {
						t.Helper()
						command := frontendCommand(compiler.config, args...)
						command.Dir = dir
						command.Env = frontendCommandEnv(compiler.config.env, dir)
						if output, err := command.CombinedOutput(); err != nil {
							t.Fatalf("%v: %v\n%s", args, err, output)
						}
					}
					run("-t", "linux/amd64", mode, "-o", "main.o", "main.go")
					object, err := elf.Open(filepath.Join(dir, "main.o"))
					if err != nil {
						t.Fatal(err)
					}
					defer object.Close()
					if object.Type != elf.ET_REL || object.Machine != elf.EM_X86_64 {
						t.Fatalf("not an amd64 relocatable: %v", object.FileHeader)
					}
					text := object.Section(".text")
					if text == nil || text.Size == 0 {
						t.Fatal("object contains no code")
					}
					symbols, err := object.Symbols()
					if err != nil {
						t.Fatal(err)
					}
					for _, name := range []string{"Add", "main"} {
						found := false
						for _, symbol := range symbols {
							if symbol.Name == name && symbol.Section != elf.SHN_UNDEF && elf.ST_TYPE(symbol.Info) == elf.STT_FUNC && elf.ST_BIND(symbol.Info) == elf.STB_GLOBAL {
								found = true
							}
						}
						if !found {
							t.Errorf("missing defined function %s", name)
						}
					}
					run("cc", "-t", "linux/amd64", "main.o", "-o", "app")
					linked, err := elf.Open(filepath.Join(dir, "app"))
					if err != nil {
						t.Fatal(err)
					}
					defer linked.Close()
					if linked.Entry == 0 || linked.Type != elf.ET_EXEC && linked.Type != elf.ET_DYN {
						t.Fatalf("not an executable image: %v", linked.FileHeader)
					}
					if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
						if output, err := exec.Command(filepath.Join(dir, "app")).CombinedOutput(); err != nil {
							t.Fatalf("execute linked Go object: %v\n%s", err, output)
						}
					}
				})
			}
		})
	}
}

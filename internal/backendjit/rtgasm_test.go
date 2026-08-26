//go:build !renvo

package backendjit

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

func TestCompilerJITProjectRTGAssembly(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	project, err := os.MkdirTemp("/var/tmp", "renvo-rtgasm-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(project) })
	assembly := []byte(`rtgasm 1
assembly {
answer(out:emitter) {
	let done = out.NewLabel()
	emitJump(out, done)
	out.Byte(0xcc)
	out.Mark(done)
	emitImmediate(out, ax, 42)
	emitReturn(out)
}
}
`)
	if err = os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.com/rtgasm\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(project, "main.go"), []byte("package main\nfunc answer() int\nfunc appMain() int { return answer() }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(project, "answer.rtgasm"), assembly, 0o600); err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "examples", "msdos", "msdos_com.rtg")
	backend := New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
		backendJITTestCacheDir, backendcompiled.Backend{})
	result := driver.CompileFromFS([]string{
		"-backend", definition, "-t", "msdos/8086", "-s", "-o", "answer.com", ".",
	}, project, filepath.Join(root, "std"), driver.OSFS{}, backend)
	if !result.Ok {
		t.Fatalf("RTGASM compile failed: %#v", result.Diagnostic)
	}
	if !bytes.Contains(result.Binary, []byte{0xb8, 0x2a, 0x00, 0xc3}) {
		t.Fatalf("COM image omits evaluated RTGASM body: %x", result.Binary)
	}
	prepared := backend.prepare("msdos/8086")
	artifactPath := filepath.Join(project, "msdos.rtgb")
	if !prepared.Ok || os.WriteFile(artifactPath, prepared.Encoded, 0o600) != nil {
		t.Fatal("could not persist prepared RTGASM backend")
	}
	artifactBackend := New(artifactPath, filepath.Join(root, "backend"), filepath.Join(root, "std"),
		backendJITTestCacheDir, backendcompiled.Backend{})
	artifactResult := driver.CompileFromFS([]string{
		"-backend", artifactPath, "-t", "msdos/8086", "-s", "-o", "answer.com", ".",
	}, project, filepath.Join(root, "std"), driver.OSFS{}, artifactBackend)
	if !artifactResult.Ok || !bytes.Contains(artifactResult.Binary, []byte{0xb8, 0x2a, 0x00, 0xc3}) {
		t.Fatalf("prepared RTGB lost RTGASM evaluation: %#v", artifactResult.Diagnostic)
	}
	invalid := []byte("rtgasm 1\nassembly { answer(out:emitter) { unknownTargetHelper(out) } }\n")
	if err = os.WriteFile(filepath.Join(project, "answer.rtgasm"), invalid, 0o600); err != nil {
		t.Fatal(err)
	}
	failed := driver.CompileFromFS([]string{
		"-backend", definition, "-t", "msdos/8086", "-s", "-o", "answer.com", ".",
	}, project, filepath.Join(root, "std"), driver.OSFS{}, backend)
	if failed.Ok || failed.Diagnostic.Code != "RENVO-RTGASM-011" || failed.Diagnostic.Path != "answer.rtgasm" {
		t.Fatalf("target-helper diagnostic = %#v", failed.Diagnostic)
	}
}

func TestCompilerJITProjectRTGAssemblyAcrossConstrainedTargets(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		definition string
		target     string
		assembly   string
		machine    []byte
		cSource    bool
	}{
		{name: "pdp11", definition: "examples/pdp11v7/pdp11_v7.rtg", target: "unixv7/pdp11", assembly: "emitImmediate(out, r0, 42)\nemitReturn(out)", machine: []byte{0xc0, 0x15, 0x2a, 0x00, 0x87, 0x00}},
		{name: "esp32c6", definition: "backends/esp32c6.rtg", target: "esp32c6/riscv32", assembly: "moveImmediate(out, a0, 42)\nret(out)", machine: []byte{0x13, 0x05, 0xa0, 0x02, 0x67, 0x80, 0x00, 0x00}},
		{name: "c89", definition: "backends/c89.rtg", target: "c89/hosted32", assembly: "c89MoveImmediate(out, c89vm32Primary, 42)\nc89Return(out)", machine: []byte{3, 0, 42, 0, 0, 0, 32}, cSource: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			project, err := os.MkdirTemp("/var/tmp", "renvo-rtgasm-target-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(project) })
			files := map[string]string{
				"go.mod":        "module example.com/rtgasm\n",
				"main.go":       "package main\nfunc answer() int\nfunc appMain() int { return answer() }\n",
				"answer.rtgasm": "rtgasm 1\nassembly {\nanswer(out:emitter) {\n" + test.assembly + "\n}\n}\n",
			}
			for name, source := range files {
				if err = os.WriteFile(filepath.Join(project, name), []byte(source), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			definition := filepath.Join(root, test.definition)
			backend := New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"), backendJITTestCacheDir, backendcompiled.Backend{})
			result := driver.CompileFromFS([]string{"-backend", definition, "-t", test.target, "-s", "-o", "image", "."}, project, filepath.Join(root, "std"), driver.OSFS{}, backend)
			if !result.Ok {
				t.Fatalf("%s RTGASM compile failed: %#v", test.target, result.Diagnostic)
			}
			binary := result.Binary
			if test.cSource {
				binary = c89GeneratedByteArray(t, binary, "rgcod")
			}
			if !bytes.Contains(binary, test.machine) {
				t.Fatalf("%s image omits %x", test.target, test.machine)
			}
		})
	}
}

func TestRTGAssemblyExamplesCompile(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		definition string
		target     string
		project    string
	}{
		{name: "msdos", definition: "examples/msdos/msdos_com.rtg", target: "msdos/8086", project: "examples/msdos"},
		{name: "pdp11", definition: "examples/pdp11v7/pdp11_v7.rtg", target: "unixv7/pdp11", project: "examples/pdp11v7"},
	} {
		t.Run(test.name, func(t *testing.T) {
			definition := filepath.Join(root, test.definition)
			backend := New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"), backendJITTestCacheDir, backendcompiled.Backend{})
			result := driver.CompileFromFS([]string{"-backend", definition, "-t", test.target, "-s", "-o", "image", "."}, filepath.Join(root, test.project), filepath.Join(root, "std"), driver.OSFS{}, backend)
			if !result.Ok {
				t.Fatalf("compile example: %#v", result.Diagnostic)
			}
		})
	}
}

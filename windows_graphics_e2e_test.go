package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWindowsGraphicsEndToEnd(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows graphics execution requires a native Windows host")
	}
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	workDir := t.TempDir()
	appDir := filepath.Join(workDir, "cmd", "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "go.mod"), []byte("module example.com/windowsgraphicse2e\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := `package main

import "graphics"

func fail(message string) {
	print("FAIL ")
	print(message)
	print("\n")
}

func decimal(value int) string {
	if value == 0 {
		return "0"
	}
	var reversed []byte
	for value > 0 {
		reversed = append(reversed, byte('0'+value%10))
		value = value / 10
	}
	out := make([]byte, len(reversed))
	for i := 0; i < len(reversed); i++ {
		out[i] = reversed[len(reversed)-1-i]
	}
	return string(out)
}

func main() {
	w := graphics.NewWindow(graphics.WindowOptions{Title: "RTG Windows graphics ✓", Width: 32, Height: 24, Hidden: true})
	if w == nil {
		err := graphics.LastWindowError()
		if err != nil {
			fail("new window: " + err.Message + " (" + decimal(err.Code) + ")")
		} else {
			fail("new window")
		}
		return
	}
	surface := w.Surface()
	surface.Clear(graphics.RGBA(231, 47, 93, 255))
	if !w.Present() {
		fail("present")
		w.Close()
		return
	}
	capture := w.ReadPixels()
	if capture == nil || capture.Width != 32 || capture.Height != 24 {
		fail("capture")
		w.Close()
		return
	}
	center := (12*capture.Stride + 16*4)
	if capture.Pixels[center] != 231 || capture.Pixels[center+1] != 47 || capture.Pixels[center+2] != 93 || capture.Pixels[center+3] != 255 {
		fail("pixels " + decimal(int(capture.Pixels[center])) + "," + decimal(int(capture.Pixels[center+1])) + "," + decimal(int(capture.Pixels[center+2])) + "," + decimal(int(capture.Pixels[center+3])))
		w.Close()
		return
	}
	w.Close()
	print("PASS\n")
}
`
	if err := os.WriteFile(filepath.Join(appDir, "main.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	backend := buildWindowsHostCompiler(t, t.TempDir())
	frontend := filepath.Join(t.TempDir(), "rtg-frontend.exe")
	buildFrontend, err := runCommandInDir(t, repoRoot, "go", "build", "-o", frontend, "./rtg/cmd/rtg")
	if err != nil {
		t.Fatalf("build frontend: %v", err)
	}
	assertWindowsCommandOK(t, "frontend build", buildFrontend)
	for _, target := range windowsE2ETargets {
		target := target
		t.Run(target.name, func(t *testing.T) {
			outputName := "graphics-" + target.name[len("windows/"):] + ".exe"
			compile := exec.Command(frontend, "-t", target.name, "-s", "-o", outputName, "./cmd/app")
			compile.Dir = workDir
			compile.Env = append(os.Environ(), "RTG_BACKEND="+backend, "RTG_STDROOT="+filepath.Join(repoRoot, "rtg", "std"))
			if output, err := compile.CombinedOutput(); err != nil {
				t.Fatalf("compile Windows graphics test: %v\n%s", err, output)
			}
			command, err := runWindowsCommand(t, workDir, filepath.Join(workDir, outputName))
			if err != nil {
				t.Fatalf("run Windows graphics test: %v", err)
			}
			if command.exitCode != 0 || command.stdout != "PASS\n" || command.stderr != "" {
				t.Fatalf("Windows graphics output mismatch: exit=%d stdout=%q stderr=%q", command.exitCode, command.stdout, command.stderr)
			}
		})
	}
}

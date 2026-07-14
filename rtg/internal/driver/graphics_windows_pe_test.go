//go:build !rtg

package driver

import (
	"debug/pe"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsGraphicsPEImports(t *testing.T) {
	repoRoot := driverRepoRoot(t)
	workDir := t.TempDir()
	appDir := filepath.Join(workDir, "cmd", "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "go.mod"), []byte("module example.com/windowsgraphicscheck\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := `package main

import "graphics"

func main() {
	w := graphics.NewWindow(graphics.WindowOptions{Title: "RTG Windows graphics", Width: 64, Height: 48, Hidden: true})
	if w == nil {
		return
	}
	w.SetTitle("RTG Windows graphics UTF-8 ✓")
	w.SetSize(80, 60)
	w.Show()
	w.Hide()
	w.RequestRepaint(graphics.R(0, 0, 10, 10))
	w.SetPointerCapture(true)
	w.SetPointerCapture(false)
	w.SetCursor(graphics.CursorIBeam)
	w.SetTimer(7, 0.01)
	w.CancelTimer(7)
	graphics.SetClipboardText("RTG ✓")
	graphics.ClipboardText()
	surface := w.Surface()
	surface.Clear(graphics.RGBA(255, 0, 0, 255))
	w.Present()
	w.ReadPixels()
	w.Poll()
	width, _ := w.Size()
	if width < 0 {
		w.Wait()
	}
	w.Close()
}
`
	if err := os.WriteFile(filepath.Join(appDir, "main.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	backend := filepath.Join(t.TempDir(), "rtgx-backend")
	build := exec.Command("go", "build", "-o", backend, ".")
	build.Dir = repoRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("backend build failed: %v\n%s", err, output)
	}
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)
	result := RunCommand(
		[]string{"rtg", "-t", "windows/amd64", "-o", "app.exe", "./cmd/app"},
		[]string{BackendEnv + "=" + backend, StdRootEnv + "=" + filepath.Join(repoRoot, "rtg", "std")},
		nil,
	)
	if !result.Ok {
		t.Fatalf("Windows graphics compilation failed: err=%d path=%q buildErr=%d arg=%q errorPath=%q at=%d", result.Error, result.ErrorPath, result.Compile.Build.Error, result.Compile.Build.ErrorArg, result.Compile.Build.ErrorPath, result.Compile.Build.ErrorAt)
	}
	image, err := pe.Open(filepath.Join(workDir, "app.exe"))
	if err != nil {
		t.Fatal(err)
	}
	defer image.Close()
	symbols, err := image.ImportedSymbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"CreateWindowExW:user32.dll",
		"GetDC:user32.dll",
		"SetClipboardData:user32.dll",
		"ChoosePixelFormat:gdi32.dll",
		"wglCreateContext:opengl32.dll",
		"glDrawPixels:opengl32.dll",
		"RtlMoveMemory:ntdll.dll",
	} {
		found := false
		for _, symbol := range symbols {
			if strings.EqualFold(symbol, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("PE imports missing %q: %v", expected, symbols)
		}
	}
}

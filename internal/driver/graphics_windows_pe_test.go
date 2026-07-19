//go:build !renvo

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
	w := graphics.NewWindow(graphics.WindowOptions{Title: "RENVO Windows graphics", Width: 64, Height: 48, Hidden: true})
	if w == nil {
		return
	}
	w.SetTitle("RENVO Windows graphics UTF-8 ✓")
	w.SetSize(80, 60)
	w.Show()
	w.Hide()
	w.RequestRepaint(graphics.R(0, 0, 10, 10))
	w.SetPointerCapture(true)
	w.SetPointerCapture(false)
	w.SetCursor(graphics.CursorIBeam)
	w.SetTimer(7, 0.01)
	w.CancelTimer(7)
	graphics.SetClipboardText("RENVO ✓")
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
	backend := filepath.Join(t.TempDir(), "renvo-backend")
	build := exec.Command("go", "build", "-o", backend, "./backend")
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
	targets := []struct {
		name         string
		arch         string
		machine      uint16
		windowSuffix string
	}{
		{name: "windows/amd64", arch: "amd64", machine: pe.IMAGE_FILE_MACHINE_AMD64, windowSuffix: "W"},
		{name: "windows/386", arch: "386", machine: pe.IMAGE_FILE_MACHINE_I386, windowSuffix: "A"},
	}
	for _, target := range targets {
		t.Run(target.arch, func(t *testing.T) {
			outputName := "app-" + target.arch + ".exe"
			result := RunCommand(
				[]string{"renvo", "-t", target.name, "-windows-gui", "-o", outputName, "./cmd/app"},
				[]string{BackendEnv + "=" + backend, StdRootEnv + "=" + filepath.Join(repoRoot, "std")},
				nil,
			)
			if !result.Ok {
				t.Fatalf("Windows graphics compilation failed: err=%d path=%q buildErr=%d arg=%q errorPath=%q at=%d", result.Error, result.ErrorPath, result.Compile.Build.Error, result.Compile.Build.ErrorArg, result.Compile.Build.ErrorPath, result.Compile.Build.ErrorAt)
			}
			image, err := pe.Open(filepath.Join(workDir, outputName))
			if err != nil {
				t.Fatal(err)
			}
			defer image.Close()
			if image.FileHeader.Machine != target.machine {
				t.Fatalf("PE machine = %#x, want %#x", image.FileHeader.Machine, target.machine)
			}
			subsystem := uint16(0)
			switch header := image.OptionalHeader.(type) {
			case *pe.OptionalHeader32:
				subsystem = header.Subsystem
			case *pe.OptionalHeader64:
				subsystem = header.Subsystem
			}
			if subsystem != pe.IMAGE_SUBSYSTEM_WINDOWS_GUI {
				t.Fatalf("PE subsystem = %d, want Windows GUI", subsystem)
			}
			if target.arch == "386" {
				header, ok := image.OptionalHeader.(*pe.OptionalHeader32)
				if !ok {
					t.Fatalf("Windows/386 image has optional header %T", image.OptionalHeader)
				}
				if header.MajorOperatingSystemVersion != 4 || header.MinorOperatingSystemVersion != 0 || header.MajorSubsystemVersion != 4 || header.MinorSubsystemVersion != 0 {
					t.Fatalf("Windows/386 PE version is OS %d.%d subsystem %d.%d, want 4.0/4.0", header.MajorOperatingSystemVersion, header.MinorOperatingSystemVersion, header.MajorSubsystemVersion, header.MinorSubsystemVersion)
				}
				if header.SizeOfImage > 0x08000000 {
					t.Fatalf("Windows/386 image reserves %#x bytes, too large for a practical Windows 98 process", header.SizeOfImage)
				}
			}
			symbols, err := image.ImportedSymbols()
			if err != nil {
				t.Fatal(err)
			}
			for _, expected := range []string{
				"CreateWindowEx" + target.windowSuffix + ":user32.dll",
				"RegisterClass" + target.windowSuffix + ":user32.dll",
				"GetModuleHandle" + target.windowSuffix + ":kernel32.dll",
				"GetDC:user32.dll",
				"SetClipboardData:user32.dll",
				"TrackMouseEvent:user32.dll",
				"ChoosePixelFormat:gdi32.dll",
				"wglCreateContext:opengl32.dll",
				"glDrawPixels:opengl32.dll",
				"RtlMoveMemory:kernel32.dll",
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
			if target.arch == "386" {
				for _, expected := range []string{
					"WideCharToMultiByte:kernel32.dll",
					"MultiByteToWideChar:kernel32.dll",
				} {
					if !windowsGraphicsHasImport(symbols, expected) {
						t.Fatalf("Windows 98 conversion import missing %q: %v", expected, symbols)
					}
				}
				for _, symbol := range symbols {
					lower := strings.ToLower(symbol)
					if strings.HasSuffix(lower, ":ntdll.dll") {
						t.Fatalf("Windows/386 image imports NT-only symbol %q", symbol)
					}
					for _, unsupported := range []string{"CreateWindowExW:user32.dll", "RegisterClassW:user32.dll", "DefWindowProcW:user32.dll", "SetWindowTextW:user32.dll", "PeekMessageW:user32.dll", "GetMessageW:user32.dll", "DispatchMessageW:user32.dll", "LoadCursorW:user32.dll", "GetModuleHandleW:kernel32.dll"} {
						if strings.EqualFold(symbol, unsupported) {
							t.Fatalf("Windows/386 image imports native-Unicode symbol %q", symbol)
						}
					}
				}
			}
		})
	}
}

func windowsGraphicsHasImport(symbols []string, expected string) bool {
	for _, symbol := range symbols {
		if strings.EqualFold(symbol, expected) {
			return true
		}
	}
	return false
}

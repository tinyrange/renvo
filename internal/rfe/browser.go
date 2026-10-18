package rfe

import (
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed browser_host.html
var browserHost string

// BrowserWorkspace keeps dependency relocation identical to native builds.
// BrowserEntry initializes without blocking; BrowserStep returns 0 while running,
// 1 on orderly shutdown, or 2 on failure. The named wrapper is the compiler's
// existing browser-step export, and is referenced from main to retain it.
func BrowserWorkspace(dir, root string, packages []*Package) error {
	if len(packages) == 0 || packages[len(packages)-1].Manifest.BrowserEntry == "" {
		return fmt.Errorf("RFE has no cooperative browser entry")
	}
	if err := Workspace(dir, root, packages); err != nil {
		return err
	}
	m := packages[len(packages)-1].Manifest
	source := fmt.Sprintf(`package main
import ("fmt"; "os"; emu "renvo.dev/internal/rfe/runtime"; entry %q)
func renvoBrowserPanic() {
    if failure := recover(); failure != nil {
        fmt.Fprintln(os.Stderr, "RENVO-RFE-BROWSER: panic:", failure)
        os.Exit(2)
    }
}
func renvoBrowserService() bool { return emu.BrowserRefill() }
func renvoBrowserStep() {
    defer renvoBrowserPanic()
    status := entry.%s()
    if status != 0 { os.Exit(status-1) }
}
func main() {
    if status := entry.%s(os.Args[1:]); status != 0 { os.Exit(status) }
    // Keep the cooperative entry reachable without executing a quantum in _start.
    if len(os.Args) == 0 { renvoBrowserStep(); renvoBrowserService() }
}
`, "renvo.dev/rfe/workspace/packages/"+m.Name, m.BrowserStep, m.BrowserEntry)
	return os.WriteFile(filepath.Join(dir, "cmd", "main.go"), []byte(source), 0644)
}

// PackageBrowserMachine uses a dedicated worker host, not the desktop GUI's
// event-record stdin contract. WASM contains all CPU and device execution.
func PackageBrowserMachine(wasm []byte) ([]byte, error) {
	if len(wasm) < 8 || string(wasm[:8]) != "\x00asm\x01\x00\x00\x00" {
		end := len(wasm)
		if end > 160 {
			end = 160
		}
		return nil, fmt.Errorf("compiler did not produce a WebAssembly module; header %q", wasm[:end])
	}
	return []byte(strings.ReplaceAll(browserHost, "@@WASM@@", base64.StdEncoding.EncodeToString(wasm))), nil
}

// BrowserSourceHash includes implementation and generated sources but excludes
// tests and sandbox artifacts. The offline checker independently reproduces it.
// The compiler executable has a separate hash in the build proof.
func BrowserSourceHash(root string) (string, error) {
	names := []string{"go.mod"}
	top, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	for _, e := range top {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
			names = append(names, e.Name())
		}
	}
	for _, dir := range []string{"backend", "cmd", "internal", "std", "emulators"} {
		err = filepath.WalkDir(filepath.Join(root, dir), func(p string, e os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if e.IsDir() {
				if e.Name() == ".git" || p == filepath.Join(root, "backend", "tests") {
					return filepath.SkipDir
				}
				return nil
			}
			if !e.Type().IsRegular() {
				return fmt.Errorf("non-regular browser source: %s", p)
			}
			ext := filepath.Ext(p)
			if strings.HasSuffix(p, "_test.go") || (ext != ".go" && ext != ".rtg" && ext != ".rbe" && ext != ".rfe" && ext != ".html") {
				return nil
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			names = append(names, filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	sort.Strings(names)
	h := sha256.New()
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			return "", err
		}
		h.Write([]byte(name))
		h.Write([]byte{0})
		h.Write(data)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

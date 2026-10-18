package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"renvo.dev/internal/rfe"
	"strconv"
)

func fileHash(name string) (string, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// Browser bootstrap executables resolve their backend beside the executable.
// Record both identities: the frontend can stay byte-identical while that
// backend changes, so its hash alone cannot establish build provenance.
type browserCompilerInputs struct {
	Frontend, FrontendHash string
	Backend, BackendHash   string
}

func browserCompilerInputHashes(compiler string) (browserCompilerInputs, error) {
	resolved, err := filepath.EvalSymlinks(compiler)
	if err != nil {
		return browserCompilerInputs{}, err
	}
	frontendHash, err := fileHash(resolved)
	if err != nil {
		return browserCompilerInputs{}, err
	}
	inputs := browserCompilerInputs{Frontend: resolved, FrontendHash: frontendHash}
	backend := filepath.Join(filepath.Dir(resolved), "renvo-backend")
	if filepath.Ext(resolved) == ".exe" {
		backend += ".exe"
	}
	if _, err := os.Stat(backend); err == nil {
		inputs.Backend, err = filepath.EvalSymlinks(backend)
		if err != nil {
			return browserCompilerInputs{}, err
		}
		inputs.BackendHash, err = fileHash(inputs.Backend)
		if err != nil {
			return browserCompilerInputs{}, err
		}
	} else if !os.IsNotExist(err) {
		return browserCompilerInputs{}, err
	}
	return inputs, nil
}

func buildBrowser(dir, root string, packages []*rfe.Package, compiler, output string, arena int) error {
	compiler, err := filepath.Abs(compiler)
	if err != nil {
		return err
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return err
	}
	// Never leave an old executable/proof looking like a successful rebuild.
	for _, p := range []string{output, output + ".json"} {
		if err = os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	sourceHash, err := rfe.BrowserSourceHash(root)
	if err != nil {
		return err
	}
	compilerInputs, err := browserCompilerInputHashes(compiler)
	if err != nil {
		return err
	}
	if err = rfe.BrowserWorkspace(dir, root, packages); err != nil {
		return err
	}
	image := filepath.Join(dir, "machine.wasm")
	// The browser target uses the WASI emitter plus an HTML host. Request its
	// raw WASM composition with browser source tags, then attach the machine
	// worker host instead of the desktop host. -emit-image is a native image
	// container, not a request for a raw WebAssembly module.
	cmd := exec.Command(compiler, "-tags", "renvo_bundle,browser", "-t", "wasi/wasm32", "-arena-size", strconv.Itoa(arena), "-o", image, "./cmd")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("Renvo browser compile: %w", err)
	}
	after, err := rfe.BrowserSourceHash(root)
	if err != nil {
		return err
	}
	compilerAfter, err := browserCompilerInputHashes(compiler)
	if err != nil {
		return err
	}
	if after != sourceHash || compilerAfter != compilerInputs {
		return fmt.Errorf("browser build inputs changed during compilation")
	}
	wasm, err := os.ReadFile(image)
	if err != nil {
		return err
	}
	html, err := rfe.PackageBrowserMachine(wasm)
	if err != nil {
		return err
	}
	if err = os.WriteFile(output, html, 0644); err != nil {
		return err
	}
	digest := sha256.Sum256(html)
	proof := struct {
		Format       int    `json:"format"`
		Machine      string `json:"machine"`
		Target       string `json:"target"`
		Arena        int    `json:"arena"`
		Sources      string `json:"sources_sha256"`
		Compiler     string `json:"compiler_sha256"`
		CompilerPath string `json:"compiler_path"`
		BackendPath  string `json:"compiler_backend_path,omitempty"`
		BackendHash  string `json:"compiler_backend_sha256,omitempty"`
		HTML         string `json:"html_sha256"`
	}{1, packages[len(packages)-1].Manifest.Name, "browser/wasm32", arena, sourceHash, compilerInputs.FrontendHash, compilerInputs.Frontend, compilerInputs.Backend, compilerInputs.BackendHash, hex.EncodeToString(digest[:])}
	data, err := json.MarshalIndent(proof, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(output+".json", append(data, '\n'), 0644); err != nil {
		return err
	}
	fmt.Printf("browser RFE: %s (%d WASM bytes, %d HTML bytes)\n", output, len(wasm), len(html))
	return nil
}

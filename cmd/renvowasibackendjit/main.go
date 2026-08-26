//go:build !renvo

// Command renvowasibackendjit resolves an RTG definition in a WASI filesystem
// and prepares a VM32 backend compiler for the browser frontend.
package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/backendjit"
	"renvo.dev/internal/linkedimage"
	"renvo.dev/internal/rtg"
)

type manifest struct {
	Name              string   `json:"name"`
	BackendTarget     string   `json:"backendTarget"`
	Output            string   `json:"output"`
	Runnable          bool     `json:"runnable,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Definition        string   `json:"definition"`
	DescriptorVersion int      `json:"descriptorVersion"`
	BackendFormat     string   `json:"backendFormat"`
}

func main() {
	inspect := flag.Bool("inspect", false, "list targets exported by the definition")
	evaluateUnit := flag.String("evaluate-unit", "", "evaluate preserved RTGASM in a compact unit")
	definition := flag.String("definition", "", "project-relative RTG definition")
	target := flag.String("target", "", "target to prepare")
	output := flag.String("o", "", "prepared VM32 backend output")
	flag.Parse()
	if *definition == "" || *inspect && (*target != "" || *output != "" || *evaluateUnit != "") ||
		!*inspect && (*target == "" || *output == "") {
		fail("usage: renvo-backend-jit -inspect -definition file.rtg | -definition file.rtg -target os/arch [-evaluate-unit input.unit] -o output")
	}
	source, err := os.ReadFile(*definition)
	if err != nil {
		fail("read definition: " + err.Error())
	}
	resolved := rtg.ResolveDefinitions(rtg.ParseImports(source, *definition, filesystemImports{}))
	if !resolved.Ok {
		failDiagnostics(resolved.Diagnostics)
	}
	if *inspect {
		manifests := make([]manifest, 0, len(resolved.Targets))
		for _, item := range resolved.Targets {
			manifests = append(manifests, targetManifest(item.Descriptor))
		}
		writeJSON(manifests)
		return
	}
	if *evaluateUnit != "" {
		descriptor, found := targetDescriptor(resolved, *target)
		if !found {
			fail("target is not exported by definition: " + *target)
		}
		unitSource, readErr := os.ReadFile(*evaluateUnit)
		if readErr != nil {
			fail("read unit: " + readErr.Error())
		}
		evaluated, result := backendjit.EvaluateRTGAssembly(unitSource, resolved, descriptor, "/std", backendcompiled.Backend{})
		if !result.Ok {
			fail(result.Diagnostic.Code + ": " + result.Diagnostic.Message)
		}
		if err = os.WriteFile(*output, evaluated, 0o644); err != nil {
			fail("write evaluated unit: " + err.Error())
		}
		return
	}
	prepared := backendjit.Prepare(backendjit.PrepareConfig{
		Definition: source, Filename: *definition, ImportLoader: filesystemImports{},
		Target: *target, StdRoot: "/std", HostTarget: "vm/vm32",
		ArenaSize: 96 * 1024 * 1024, Bootstrap: backendcompiled.Backend{},
	})
	if !prepared.Ok {
		fail(prepared.Diagnostic.Message)
	}
	image, err := linkedimage.Decode(prepared.Artifact.Payload)
	if err != nil || image.Target != "vm/vm32" || len(image.Native) == 0 {
		fail("prepared backend did not contain a VM32 compiler")
	}
	if err = os.WriteFile(*output, image.Native, 0o644); err != nil {
		fail("write backend: " + err.Error())
	}
	writeJSON(targetManifest(prepared.Artifact.Descriptor))
}

func targetDescriptor(resolved rtg.ResolveResult, name string) (rtg.TargetDescriptor, bool) {
	for _, target := range resolved.Targets {
		if target.Descriptor.Name == name {
			return target.Descriptor, true
		}
	}
	return rtg.TargetDescriptor{}, false
}

type filesystemImports struct{}

func (filesystemImports) LoadImport(importingFilename string, importPath string) rtg.ImportSource {
	path := filepath.Clean(filepath.Join(filepath.Dir(importingFilename), importPath))
	source, err := os.ReadFile(path)
	return rtg.ImportSource{Source: source, Filename: path, Ok: err == nil}
}

func targetManifest(descriptor rtg.TargetDescriptor) manifest {
	return manifest{
		Name: descriptor.Name, BackendTarget: descriptor.Name,
		Output:   outputName(descriptor.Name, descriptor.OutputKind),
		Runnable: descriptor.OutputKind == "wasm" && descriptor.Name == "wasi/wasm32",
		Tags:     descriptor.BuildTags, Definition: hex.EncodeToString(descriptor.Definition[:]),
		DescriptorVersion: descriptor.Version, BackendFormat: "vm32",
	}
}

func outputName(target string, kind string) string {
	if kind == "wasm" || kind == "html-wasm" {
		return "app.wasm"
	}
	if kind == "rnvm" {
		return "app.rnvb"
	}
	if kind == "dos-com" {
		return "app.com"
	}
	if kind == "dos-mz" {
		return "app.exe"
	}
	if strings.HasPrefix(target, "windows/") {
		return "app.exe"
	}
	if kind == "elf" {
		return "app.elf"
	}
	return "app"
}

func writeJSON(value any) {
	if err := json.NewEncoder(os.Stdout).Encode(value); err != nil {
		fail("encode manifest: " + err.Error())
	}
}

func failDiagnostics(diagnostics []rtg.Diagnostic) {
	for _, diagnostic := range diagnostics {
		fmt.Fprintf(os.Stderr, "%s:%d:%d: %s: %s\n", diagnostic.Filename,
			diagnostic.Span.Start.Line, diagnostic.Span.Start.Column,
			diagnostic.Code, diagnostic.Message)
	}
	os.Exit(1)
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, "renvo-backend-jit:", message)
	os.Exit(1)
}

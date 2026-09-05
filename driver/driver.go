package driver

import (
	"fmt"
	"strings"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

type SourceFS = driver.SourceFS
type Diagnostic = driver.Diagnostic
type DirEntry = driver.DirEntry

type Request struct {
	Filesystem SourceFS
	Input      []string
	Target     string
	ArenaSize  uint64
}

type Result struct {
	Ok         bool
	Diagnostic Diagnostic
	Binary     []byte
}

func formatArguments(request *Request) []string {
	args := make([]string, 0, len(request.Input)+4)
	args = append(args, "-o", "output.bin")
	args = append(args, "-t", request.Target)
	args = append(args, "-arena-size", fmt.Sprintf("%d", request.ArenaSize))
	args = append(args, request.Input...)
	return args
}

type overlayFS struct {
	base SourceFS
	std  SourceFS
}

// PathExists implements [driver.SourceFS].
func (o *overlayFS) PathExists(path string) bool {
	if ok := strings.HasPrefix(path, "std/"); ok {
		return o.std.PathExists(path)
	}

	// Check if the path exists in the base filesystem.
	return o.base.PathExists(path)
}

// ReadDir implements [driver.SourceFS].
func (o *overlayFS) ReadDir(path string) ([]driver.DirEntry, bool) {
	if ok := strings.HasPrefix(path, "std/"); ok {
		if o.std == nil {
			return nil, false
		}

		return o.std.ReadDir(path)
	}

	// Read the directory from the base filesystem.
	return o.base.ReadDir(path)
}

// ReadFile implements [driver.SourceFS].
func (o *overlayFS) ReadFile(path string) ([]byte, bool) {
	if ok := strings.HasPrefix(path, "std/"); ok {
		if o.std == nil {
			return nil, false
		}

		return o.std.ReadFile(path)
	}

	if path == "go.mod" {
		return []byte("module main\n"), true
	}

	// Read the file from the base filesystem.
	return o.base.ReadFile(path)
}

var (
	_ SourceFS = &overlayFS{}
)

func Compile(request *Request) (*Result, error) {
	if request.Target == "" {
		return nil, fmt.Errorf("target must be specified")
	}
	if len(request.Input) == 0 {
		return nil, fmt.Errorf("at least one input file must be specified")
	}

	result := driver.CompileFromFSWithModuleCache(
		formatArguments(request),
		".",
		"std/",
		".",
		&overlayFS{
			base: request.Filesystem,
			std:  BundledSourceFS(),
		},
		backendcompiled.Backend{},
	)

	return &Result{
		Ok:         result.Ok,
		Diagnostic: result.Diagnostic,
		Binary:     result.Binary,
	}, nil
}

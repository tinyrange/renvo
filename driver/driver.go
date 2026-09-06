package driver

import (
	"fmt"
	"path"
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
	if request.ArenaSize != 0 {
		args = append(args, "-arena-size", fmt.Sprintf("%d", request.ArenaSize))
	}
	args = append(args, request.Input...)
	return args
}

type overlayFS struct {
	base SourceFS
	std  SourceFS
}

// PathExists implements [driver.SourceFS].
func (o *overlayFS) PathExists(path string) bool {
	if isStdPath(path) {
		return o.std.PathExists(path)
	}

	// Check if the path exists in the base filesystem.
	return o.base.PathExists(path) || o.std != nil && o.std.PathExists(path)
}

// ReadDir implements [driver.SourceFS].
func (o *overlayFS) ReadDir(path string) ([]driver.DirEntry, bool) {
	if isStdPath(path) {
		if o.std == nil {
			return nil, false
		}

		return o.std.ReadDir(path)
	}

	// Read the directory from the base filesystem.
	if entries, ok := o.base.ReadDir(path); ok {
		return entries, true
	}
	if o.std != nil {
		return o.std.ReadDir(path)
	}
	return nil, false
}

// ReadFile implements [driver.SourceFS].
func (o *overlayFS) ReadFile(path string) ([]byte, bool) {
	if isStdPath(path) {
		if o.std == nil {
			return nil, false
		}

		return o.std.ReadFile(path)
	}

	if path == "go.mod" {
		if data, ok := o.base.ReadFile(path); ok {
			return data, true
		}
		return []byte("module main\n"), true
	}

	// Read the file from the base filesystem.
	if data, ok := o.base.ReadFile(path); ok {
		return data, true
	}
	if o.std != nil {
		return o.std.ReadFile(path)
	}
	return nil, false
}

var (
	_ SourceFS = &overlayFS{}
)

func Compile(request *Request) (*Result, error) {
	if request == nil || request.Filesystem == nil {
		return nil, fmt.Errorf("source filesystem must be specified")
	}
	if request.Target == "" {
		return nil, fmt.Errorf("target must be specified")
	}
	if len(request.Input) == 0 {
		return nil, fmt.Errorf("at least one input file must be specified")
	}

	local := *request
	local.Input = append([]string(nil), request.Input...)
	workDir := "."
	if len(local.Input) == 1 {
		name := path.Clean(local.Input[0])
		if _, ok := request.Filesystem.ReadDir(name); ok {
			workDir, local.Input[0] = name, "."
		} else {
			workDir, local.Input[0] = path.Dir(name), path.Base(name)
		}
	}
	result := driver.CompileFromFSWithModuleCache(
		formatArguments(&local),
		workDir,
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

func isStdPath(name string) bool {
	name = strings.TrimPrefix(path.Clean(name), "/")
	return name == "std" || strings.HasPrefix(name, "std/")
}

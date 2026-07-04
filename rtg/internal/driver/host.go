//go:build !rtg

package driver

import (
	"os"

	"j5.nz/rtg/rtg/internal/load"
)

const (
	HostOK = iota
	HostErrWorkDir
	HostErrCompile
	HostErrWrite
)

type OSFS struct{}

type HostResult struct {
	Compile   CompileResult
	Ok        bool
	Error     int
	ErrorPath string
}

func (OSFS) ReadDir(path string) ([]DirEntry, bool) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, false
	}
	out := make([]DirEntry, 0, len(entries))
	for i := 0; i < len(entries); i++ {
		out = append(out, DirEntry{Name: entries[i].Name(), IsDir: entries[i].IsDir()})
	}
	return out, true
}

func (OSFS) ReadFile(path string) ([]byte, bool) {
	data, err := os.ReadFile(path)
	return data, err == nil
}

func CompileAndWrite(args []string, stdRoot string, backend Backend) HostResult {
	result := HostResult{Ok: true, Error: HostOK}
	workDir, err := os.Getwd()
	if err != nil {
		return hostFail(result, HostErrWorkDir, "")
	}
	compiled := CompileFromFS(args, load.CleanPath(workDir), stdRoot, OSFS{}, backend)
	result.Compile = compiled
	if !compiled.Ok {
		return hostFail(result, HostErrCompile, "")
	}
	output := compiled.Build.Options.Output
	if output == "-" {
		_, err = os.Stdout.Write(compiled.Binary)
	} else {
		err = os.WriteFile(output, compiled.Binary, 0o755)
	}
	if err != nil {
		return hostFail(result, HostErrWrite, output)
	}
	return result
}

func hostFail(result HostResult, err int, path string) HostResult {
	result.Ok = false
	result.Error = err
	result.ErrorPath = path
	return result
}

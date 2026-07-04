//go:build !rtg

package driver

import (
	"os"

	"j5.nz/rtg/rtg/internal/load"
)

const (
	HostOK = iota
	HostErrWorkDir
	HostErrBackend
	HostErrCompile
	HostErrWrite
)

const BackendEnv = "RTG_BACKEND"
const StdRootEnv = "RTG_STDROOT"
const DefaultStdRoot = "/std"

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

func RunCommand(args []string, env []string, backend Backend) HostResult {
	if len(args) > 0 {
		args = args[1:]
	}
	return CompileAndWriteWithEnv(args, env, backend)
}

func CompileAndWriteWithEnv(args []string, env []string, backend Backend) HostResult {
	if backend == nil {
		commandBackend, ok := CommandBackendFromEnv(env)
		if !ok {
			return hostFail(HostResult{}, HostErrBackend, "")
		}
		backend = commandBackend
	}
	return CompileAndWrite(args, StdRootFromEnv(env), backend)
}

func CommandBackendFromEnv(env []string) (CommandBackend, bool) {
	path := EnvValue(env, BackendEnv)
	if path == "" {
		return CommandBackend{}, false
	}
	return CommandBackend{Path: path}, true
}

func StdRootFromEnv(env []string) string {
	root := EnvValue(env, StdRootEnv)
	if root == "" {
		return DefaultStdRoot
	}
	return root
}

func EnvValue(env []string, key string) string {
	prefix := key + "="
	for i := 0; i < len(env); i++ {
		item := env[i]
		if len(item) < len(prefix) {
			continue
		}
		matched := true
		for j := 0; j < len(prefix); j++ {
			if item[j] != prefix[j] {
				matched = false
				break
			}
		}
		if matched {
			return item[len(prefix):]
		}
	}
	return ""
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

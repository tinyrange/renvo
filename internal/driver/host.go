//go:build !renvo

package driver

import (
	"os"

	"renvo.dev/internal/load"
)

const (
	HostOK = iota
	HostErrWorkDir
	HostErrBackend
	HostErrCompile
	HostErrWrite
)

const BackendEnv = "RENVO_BACKEND"
const StdRootEnv = "RENVO_STDROOT"
const ModuleCacheEnv = "RENVO_MODCACHE"
const DefaultStdRoot = "/std"

type OSFS struct{}

type HostResult struct {
	Compile   CompileResult
	Ok        bool
	Error     int
	ErrorPath string
}

func (OSFS) ReadDir(path string) ([]DirEntry, bool) {
	if entries, ok := bundledStdReadDir(path); ok {
		return entries, true
	}
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
	if data, ok := bundledStdReadFile(path); ok {
		return data, true
	}
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
	options := ParseOptions(args)
	if backend == nil && options.Ok && !options.EmitUnit {
		commandBackend, ok := CommandBackendFromEnv(env)
		if !ok {
			return hostFail(HostResult{}, HostErrBackend, "")
		}
		backend = commandBackend
	}
	return compileAndWrite(args, StdRootFromEnv(env), EnvValue(env, ModuleCacheEnv), backend)
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
	return compileAndWrite(args, stdRoot, "", backend)
}

func compileAndWrite(args []string, stdRoot string, moduleCache string, backend Backend) HostResult {
	result := HostResult{Ok: true, Error: HostOK}
	workDir, err := os.Getwd()
	if err != nil {
		return hostFail(result, HostErrWorkDir, "")
	}
	compiled := CompileFromFSWithModuleCache(args, load.CleanPath(workDir), stdRoot, moduleCache, OSFS{}, backend)
	result.Compile = compiled
	if !compiled.Ok {
		return hostFail(result, HostErrCompile, "")
	}
	output := compiled.Build.Options.Output
	mode := os.FileMode(0o755)
	if compiled.Build.Options.EmitUnit {
		mode = 0o644
	}
	if output == "-" {
		_, err = os.Stdout.Write(compiled.Binary)
	} else {
		err = os.WriteFile(output, compiled.Binary, mode)
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

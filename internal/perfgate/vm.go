package perfgate

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"renvo.dev/std/vm"
)

type VMRequest struct {
	Root, Compiler, Output, Stats string
	Input                         string
	Args                          []string
	Memory                        int
	Steps                         int64
}
type VMStats struct{ Steps, Memory int }

// ExecuteVM runs in the measured OS child, including interpreter work and
// source loading. Only the requested output is copied out of the isolated VFS.
func ExecuteVM(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var request VMRequest
	if err = json.Unmarshal(data, &request); err != nil {
		return err
	}
	if request.Steps > int64(^uint(0)>>1) {
		return fmt.Errorf("VM self-hosting instruction accounting requires a 64-bit host")
	}
	image, err := os.ReadFile(request.Compiler)
	if err != nil {
		return err
	}
	var files []vm.File
	if request.Input != "" {
		data, err := os.ReadFile(request.Input)
		if err != nil {
			return err
		}
		files = append(files, vm.File{Name: "/input.unit", Data: data, Mode: 0644})
	} else if request.Output != "" {
		err = filepath.WalkDir(request.Root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			relative, err := filepath.Rel(request.Root, path)
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if relative != "." && (entry.Name() == ".git" || entry.Name() == "sandbox" || entry.Name() == "local" || entry.Name() == ".renvo") {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			ext := filepath.Ext(path)
			bundleAsset := false
			for _, prefix := range []string{"std/", "forms/", "device/", "x/", "libc/"} {
				bundleAsset = bundleAsset || strings.HasPrefix(filepath.ToSlash(relative), prefix)
			}
			if !bundleAsset && ext != ".go" && ext != ".rtg" && ext != ".rbe" && entry.Name() != "go.mod" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			files = append(files, vm.File{Name: "/workspace/" + filepath.ToSlash(relative), Data: data, Mode: 0644})
			return nil
		})
		if err != nil {
			return err
		}
		// Smoke fixtures live in sandbox and are intentionally selected explicitly.
		for _, arg := range request.Args {
			if strings.HasPrefix(arg, "./sandbox/") && strings.HasSuffix(arg, ".go") {
				data, err := os.ReadFile(filepath.Join(request.Root, arg))
				if err != nil {
					return err
				}
				files = append(files, vm.File{Name: "/workspace/" + strings.TrimPrefix(arg, "./"), Data: data, Mode: 0644})
			}
		}
	}
	args := append([]string{"renvo"}, request.Args...)
	for i, arg := range args {
		if arg == request.Input && request.Input != "" {
			args[i] = "/input.unit"
		}
		if arg == request.Output && request.Output != "" {
			args[i] = "/output/artifact"
		}
	}
	result := vm.RunConfig(image, vm.Config{Limits: vm.Limits{Steps: int(request.Steps), Memory: request.Memory}, Args: args, Env: []string{"PWD=/workspace", "RENVO_STDROOT=/workspace/std", "PATH=/vm"}, Files: files})
	_, _ = os.Stdout.Write(result.Output)
	_, _ = os.Stderr.Write(result.Stderr)
	if result.Trap != vm.TrapNone || result.ExitCode != 0 {
		return fmt.Errorf("VM exit=%d trap=%d pc=%d steps=%d memory=%d", result.ExitCode, result.Trap, result.TrapPC, result.Steps, result.PeakMemory)
	}
	if request.Stats != "" {
		data, _ := json.Marshal(VMStats{Steps: result.Steps, Memory: result.PeakMemory})
		if err = os.WriteFile(request.Stats, data, 0600); err != nil {
			return err
		}
	}
	if request.Output != "" {
		for _, file := range result.Files {
			if file.Name == "/output/artifact" {
				return os.WriteFile(request.Output, file.Data, 0755)
			}
		}
		return fmt.Errorf("VM compiler did not write its requested output")
	}
	return nil
}

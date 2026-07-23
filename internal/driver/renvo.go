//go:build renvo

package driver

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/backendbridge"
	"renvo.dev/internal/fronttrace"
	"renvo.dev/internal/load"
	"renvo.dev/std/os"
)

type RenvoFS struct{}

func (RenvoFS) PathExists(path string) bool { return renvoPathExists(path) }

const renvoCommandDiagnosticCapacity = 8192

// Keep one capture buffer outside per-build arena marks. A failed frontend can
// copy its diagnostic here before resetting all of its transient allocations.
var renvoCommandDiagnosticBuffer []byte

func RunRenvoCommand(args []string, env []string) int {
	status, output := runRenvoCommand(args, env)
	if output != "" {
		print(output)
	}
	return status
}

// RunRenvoCommandCapture runs the embedded compiler without sending its
// diagnostic to the parent terminal. GUI callers use the returned text in
// their own output surface.
func RunRenvoCommandCapture(args []string, env []string) (int, string) {
	status, output := runRenvoCommand(args, env)
	return status, output
}

func runRenvoCommand(args []string, env []string) (int, string) {
	if len(args) > 1 && args[1] == "run" {
		return runRenvoScript(args, env)
	}
	if CommandHelpRequested(args) {
		return 0, HelpText
	}
	if len(renvoCommandDiagnosticBuffer) == 0 {
		renvoCommandDiagnosticBuffer = make([]byte, renvoCommandDiagnosticCapacity)
	}
	commandArgs := args
	if len(commandArgs) > 0 {
		commandArgs = commandArgs[1:]
	}
	resetArena := renvoFrontendCanResetArena()
	fronttrace.SetEnabled(renvoEnvValue(env, "RENVO_FRONTEND_TRACE") == "1")
	fronttrace.Event("source")
	mark := 0
	if resetArena {
		mark = arena.Mark()
	}
	built := buildFromFSOneShotCompactWithModuleCache(commandArgs, renvoWorkDir(env), renvoStdRoot(args, env), renvoModuleCache(env), RenvoFS{})
	fronttrace.Event("frontend complete")
	if !built.Ok {
		return finishRenvoCommandFailure(renvoCommandDiagnosticBuffer, built.Diagnostic, resetArena, mark)
	}
	unit := built.Unit
	target := built.Options.Target
	output := built.Options.Output
	arenaSize := backendArenaSize(target, built.Options.Tags, built.Options.ArenaSize)
	if built.Options.EmitUnit {
		if output == "-" {
			print(string(unit))
		} else if os.WriteFile(output, unit, 0644) != nil {
			return finishRenvoCommandFailure(renvoCommandDiagnosticBuffer, Diagnostic{Phase: "unit", Code: "RENVO-UNIT-002", Message: "failed to write linked unit"}, resetArena, mark)
		}
		if resetArena {
			arena.Reset(mark)
		}
		return 0, ""
	}
	persistMark := 0
	if resetArena {
		persistMark = arena.PersistMark()
		unit = arena.PersistBytes(unit)
		target = arena.PersistString(target)
		output = arena.PersistString(output)
		backendMark := mark
		remainder := backendMark % 4096
		if remainder != 0 {
			backendMark += 4096 - remainder
		}
		arena.Reset(backendMark)
	}
	virtualTarget := target
	target = backendTargetForOptions(target, built.Options.Mode)
	ok := backendbridge.CompileUnitToOutputStripEnv(unit, target, output, built.Options.Strip, built.Options.WindowsGUI, built.Options.EmitImage, arenaSize, built.Options.ModuleLicense, args, env)
	if ok && virtualTarget == "browser/wasm32" && !built.Options.EmitImage {
		wasm, readErr := os.ReadFile(output)
		if readErr != nil || os.WriteFile(output, PackageBrowserHTML(wasm), 0644) != nil {
			ok = false
		}
	}
	if resetArena {
		arena.PersistReset(persistMark)
	}
	if !ok {
		return finishRenvoCommandFailure(renvoCommandDiagnosticBuffer, Diagnostic{Phase: "backend", Code: "RENVO-BACKEND-001", Message: "backend compilation failed"}, false, 0)
	}
	return 0, ""
}

func finishRenvoCommandFailure(buffer []byte, diagnostic Diagnostic, resetArena bool, mark int) (int, string) {
	formatted := FormatDiagnostic(diagnostic)
	used := len(formatted)
	if used > renvoCommandDiagnosticCapacity {
		used = renvoCommandDiagnosticCapacity
	}
	for i := 0; i < used; i++ {
		buffer[i] = formatted[i]
	}
	if used == renvoCommandDiagnosticCapacity && used > 0 {
		buffer[used-1] = '\n'
	}
	if resetArena {
		arena.Reset(mark)
	}
	return 1, string(buffer[:used])
}

func renvoWorkDir(env []string) string {
	value := renvoEnvValue(env, "PWD")
	if value != "" {
		return value
	}
	return "."
}

func renvoStdRoot(args []string, env []string) string {
	value := renvoEnvValue(env, "RENVO_STDROOT")
	if value != "" {
		return value
	}
	if renvoBundledStdEnabled {
		return "/std"
	}
	bundled := renvoBundledStdRoot(args)
	if bundled != "" {
		return bundled
	}
	return "/std"
}

func renvoModuleCache(env []string) string {
	value := renvoEnvValue(env, "RENVO_MODCACHE")
	if value == "" && renvoBundledStdEnabled {
		return "/modules"
	}
	return value
}

func renvoEnvValue(env []string, key string) string {
	for i := 0; i < len(env); i++ {
		item := env[i]
		if len(item) <= len(key) || item[len(key)] != '=' {
			continue
		}
		matched := true
		for j := 0; j < len(key); j++ {
			if item[j] != key[j] {
				matched = false
				break
			}
		}
		if matched {
			return item[len(key)+1:]
		}
	}
	return ""
}

func renvoBundledStdRoot(args []string) string {
	dir := renvoExecutableDir(args)
	if dir == "" {
		return ""
	}
	path := load.JoinPath(dir, "std")
	if renvoPathExists(path) {
		return path
	}
	path = load.JoinPath(load.JoinPath(dir, ".."), "std")
	if renvoPathExists(path) {
		return path
	}
	path = load.JoinPath(load.JoinPath(load.JoinPath(dir, ".."), "share"), "renvo")
	path = load.JoinPath(path, "std")
	if renvoPathExists(path) {
		return path
	}
	return ""
}

func renvoExecutableDir(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return load.DirPath(args[0])
}

func renvoPathExists(path string) bool {
	fd := open(renvoPathCString(path), 0)
	if fd < 0 {
		return false
	}
	close(fd)
	return true
}

func (fs RenvoFS) ReadFile(path string) ([]byte, bool) {
	if data, bundled := bundledStdReadFile(path); bundled {
		return data, true
	}
	fd := open(renvoPathCString(path), 0)
	if fd < 0 {
		return nil, false
	}
	// Most compiler source files fit in 32 KiB. Starting there avoids repeated
	// arena copies while loading a package, and the transient build releases the
	// modest unused tail after lowering.
	out := make([]byte, 32768)
	used := 0
	for {
		if used == len(out) {
			next := make([]byte, len(out)*2)
			copy(next, out)
			// Arena allocation does not garbage-collect superseded growth
			// buffers. Release their complete pages as soon as the copy is
			// finished so one unusually large compiler source file does not
			// determine the frontend's peak resident memory.
			arena.DiscardBytes(out)
			out = next
		}
		n := read(fd, out[used:], -1)
		if n < 0 {
			close(fd)
			return nil, false
		}
		if n == 0 {
			break
		}
		used += n
	}
	close(fd)
	return out[:used], true
}

func (fs RenvoFS) ReadDir(path string) ([]DirEntry, bool) {
	if entries, bundled := bundledStdReadDir(path); bundled {
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

func renvoPrintInt(value int) {
	if value == 0 {
		print("0")
		return
	}
	if value < 0 {
		print("-")
		value = -value
	}
	var digits []byte
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value = value / 10
	}
	for i := len(digits) - 1; i >= 0; i-- {
		print(string(digits[i : i+1]))
	}
}

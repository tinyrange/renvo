//go:build rtg

package driver

import (
	"j5.nz/rtg/rtg/internal/arena"
	"j5.nz/rtg/rtg/internal/backendbridge"
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/std/os"
)

type RTGFS struct{}

const rtgCommandDiagnosticCapacity = 8192

// Keep one capture buffer outside per-build arena marks. A failed frontend can
// copy its diagnostic here before resetting all of its transient allocations.
var rtgCommandDiagnosticBuffer []byte

func RunRTGCommand(args []string, env []string) int {
	status, output := runRTGCommand(args, env)
	if output != "" {
		print(output)
	}
	return status
}

// RunRTGCommandCapture runs the embedded compiler without sending its
// diagnostic to the parent terminal. GUI callers use the returned text in
// their own output surface.
func RunRTGCommandCapture(args []string, env []string) (int, string) {
	status, output := runRTGCommand(args, env)
	return status, output
}

func runRTGCommand(args []string, env []string) (int, string) {
	if CommandHelpRequested(args) {
		return 0, HelpText
	}
	if len(rtgCommandDiagnosticBuffer) == 0 {
		rtgCommandDiagnosticBuffer = make([]byte, rtgCommandDiagnosticCapacity)
	}
	commandArgs := args
	if len(commandArgs) > 0 {
		commandArgs = commandArgs[1:]
	}
	resetArena := rtgFrontendCanResetArena()
	mark := 0
	if resetArena {
		mark = arena.Mark()
	}
	built := buildFromFSCompactWithModuleCache(commandArgs, rtgWorkDir(env), rtgStdRoot(args, env), rtgEnvValue(env, "RTG_MODCACHE"), RTGFS{})
	if !built.Ok {
		return finishRTGCommandFailure(rtgCommandDiagnosticBuffer, built.Diagnostic, resetArena, mark)
	}
	unit := built.Unit
	target := built.Options.Target
	output := built.Options.Output
	strip := built.Options.Strip
	windowsGUI := built.Options.WindowsGUI
	arenaSize := built.Options.ArenaSize
	if built.Options.EmitUnit {
		if output == "-" {
			print(string(unit))
		} else if os.WriteFile(output, unit, 0644) != nil {
			return finishRTGCommandFailure(rtgCommandDiagnosticBuffer, Diagnostic{Phase: "unit", Code: "RTG-UNIT-002", Message: "failed to write linked unit"}, resetArena, mark)
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
			backendMark = backendMark + 4096 - remainder
		}
		arena.Reset(backendMark)
	}
	ok := backendbridge.CompileUnitToOutputStripEnv(unit, target, output, strip, windowsGUI, arenaSize, args, env)
	if resetArena {
		arena.PersistReset(persistMark)
	}
	if !ok {
		return finishRTGCommandFailure(rtgCommandDiagnosticBuffer, Diagnostic{Phase: "backend", Code: "RTG-BACKEND-001", Message: "backend compilation failed"}, false, 0)
	}
	return 0, ""
}

func finishRTGCommandFailure(buffer []byte, diagnostic Diagnostic, resetArena bool, mark int) (int, string) {
	formatted := FormatDiagnostic(diagnostic)
	used := len(formatted)
	if used > rtgCommandDiagnosticCapacity {
		used = rtgCommandDiagnosticCapacity
	}
	for i := 0; i < used; i++ {
		buffer[i] = formatted[i]
	}
	if used == rtgCommandDiagnosticCapacity && used > 0 {
		buffer[used-1] = '\n'
	}
	if resetArena {
		arena.Reset(mark)
	}
	return 1, string(buffer[:used])
}

func rtgWorkDir(env []string) string {
	value := rtgEnvValue(env, "PWD")
	if value != "" {
		return value
	}
	return "."
}

func rtgStdRoot(args []string, env []string) string {
	value := rtgEnvValue(env, "RTG_STDROOT")
	if value != "" {
		return value
	}
	if rtgBundledStdEnabled {
		return "/std"
	}
	bundled := rtgBundledStdRoot(args)
	if bundled != "" {
		return bundled
	}
	return "/std"
}

func rtgEnvValue(env []string, key string) string {
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

func rtgBundledStdRoot(args []string) string {
	dir := rtgExecutableDir(args)
	if dir == "" {
		return ""
	}
	path := load.JoinPath(dir, "std")
	if rtgPathExists(path) {
		return path
	}
	path = load.JoinPath(load.JoinPath(dir, ".."), "std")
	if rtgPathExists(path) {
		return path
	}
	path = load.JoinPath(load.JoinPath(load.JoinPath(dir, ".."), "share"), "rtg")
	path = load.JoinPath(path, "std")
	if rtgPathExists(path) {
		return path
	}
	return ""
}

func rtgExecutableDir(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return load.DirPath(args[0])
}

func rtgPathExists(path string) bool {
	fd := open(rtgPathCString(path), 0)
	if fd < 0 {
		return false
	}
	close(fd)
	return true
}

func (fs RTGFS) ReadFile(path string) ([]byte, bool) {
	if data, bundled := bundledStdReadFile(path); bundled {
		return data, true
	}
	fd := open(rtgPathCString(path), 0)
	if fd < 0 {
		return nil, false
	}
	if !rtgFrontendCanResetArena() {
		out := make([]byte, 4096)
		used := 0
		for {
			if used == len(out) {
				next := make([]byte, len(out)*2)
				copy(next, out)
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
	arenaStart := arena.Mark()
	probe := make([]byte, 4096)
	size := 0
	for {
		n := read(fd, probe, -1)
		if n < 0 {
			close(fd)
			arena.Reset(arenaStart)
			return nil, false
		}
		if n == 0 {
			break
		}
		size += n
	}
	close(fd)
	arena.Reset(arenaStart)
	fd = open(rtgPathCString(path), 0)
	if fd < 0 {
		return nil, false
	}
	out := make([]byte, size+1)
	used := 0
	for {
		if used == len(out) {
			next := make([]byte, len(out)*2)
			copy(next, out)
			out = next
		}
		n := read(fd, out[used:], -1)
		if n < 0 {
			close(fd)
			arena.Reset(arenaStart)
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

func (fs RTGFS) ReadDir(path string) ([]DirEntry, bool) {
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

func rtgPrintInt(value int) {
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

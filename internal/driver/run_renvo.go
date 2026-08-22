//go:build renvo

package driver

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/backendbridge"
	"renvo.dev/internal/linkedimage"
)

var renvoRunJITCall func(int, int, int, int, int, int) int

func SetRunJITCall(handler func(int, int, int, int, int, int) int) {
	renvoRunJITCall = handler
}

func runRenvoScript(args []string, env []string) (int, string) {
	compileArgs, programArgs, script, parseError := parseRenvoRunCommand(args)
	if parseError == "help" {
		return 0, RunHelpText
	}
	if parseError != "" {
		return 1, "renvo: error RENVO-RUN-001 (options): " + parseError + "\n" + RunHelpText
	}
	target := renvoRunTarget()
	if target == "" {
		return 1, "renvo: error RENVO-RUN-002 (runtime): scripts cannot execute on this host target\n"
	}
	compileArgs = append(compileArgs, "-script", "-emit-image", "-t", target, "-o", "-")
	resetArena := renvoFrontendCanResetArena()
	mark := 0
	if resetArena {
		mark = arena.Mark()
	}
	built := buildFromFSOneShotCompactWithModuleCache(compileArgs, renvoWorkDir(env), renvoStdRoot(args, env), renvoModuleCache(env), RenvoFS{})
	if !built.Ok {
		return finishRenvoCommandFailure(renvoCommandDiagnosticBuffer[:], built.Diagnostic, resetArena, mark)
	}
	unit := built.Unit
	arenaSize := backendArenaSize(target, built.Options.Tags, built.Options.ArenaSize, ModeExecutable)
	moduleLicense := built.Options.ModuleLicense
	persistMark := 0
	backendMark := 0
	if resetArena {
		persistMark = arena.PersistMark()
		// The compact pipeline leaves the linked unit as its final low-arena
		// allocation. Transfer that allocation just as the ordinary command path
		// does; copying a complete self-hosted frontend unit here needlessly
		// doubles its peak arena use before the backend can start.
		unit = arena.PersistLastBytes(unit, mark)
		target = arena.PersistString(target)
		moduleLicense = arena.PersistString(moduleLicense)
		backendMark = mark
		remainder := backendMark % 4096
		if remainder != 0 {
			backendMark += 4096 - remainder
		}
		arena.Reset(backendMark)
	}
	image, compiled := backendbridge.CompileUnitToImage(unit, target, true, arenaSize, moduleLicense)
	if !compiled {
		if resetArena {
			arena.PersistReset(persistMark)
		}
		return finishRenvoCommandFailure(renvoCommandDiagnosticBuffer[:], Diagnostic{Phase: "backend", Code: "RENVO-BACKEND-001", Message: "backend compilation failed"}, false, 0)
	}
	if resetArena {
		// The in-memory backend leaves the RNVI image as its final allocation.
		// Retain that small result and release compiler scratch before the native
		// loader allocates segment metadata and argument vectors.
		image = arena.PersistLastBytes(image, backendMark)
		arena.Reset(backendMark)
	}
	imageTarget, _, native, imageOK := linkedimage.Payload(image)
	if !imageOK || imageTarget != renvoRunTargetID() || len(native) == 0 {
		if resetArena {
			arena.PersistReset(persistMark)
		}
		return 1, "renvo: error RENVO-RUN-003 (image): backend returned an invalid host linked image\n"
	}
	exitCode := RunNativeLinkedImage(native, script, programArgs, env)
	if exitCode < 0 {
		if resetArena {
			arena.PersistReset(persistMark)
		}
		return 1, "renvo: error RENVO-RUN-004 (runtime): failed to execute linked image\n"
	}
	if resetArena {
		arena.PersistReset(persistMark)
	}
	return exitCode, ""
}

func parseRenvoRunCommand(args []string) ([]string, []string, string, string) {
	if len(args) < 3 {
		return nil, nil, "", "missing script"
	}
	var programArgs []string
	compileEnd := len(args)
	for i := 2; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			compileEnd = i
			programArgs = args[i+1:]
			break
		}
		if arg == "--help" || arg == "-h" {
			return nil, nil, "", "help"
		}
	}
	if compileEnd <= 2 {
		return nil, nil, "", "missing script"
	}
	script := args[compileEnd-1]
	if !optionArgIsGoFile(script) {
		return nil, nil, "", "script must be one .go file: " + script
	}
	var compileArgs []string
	for i := 2; i < compileEnd; i++ {
		compileArgs = append(compileArgs, args[i])
	}
	return compileArgs, programArgs, script, ""
}

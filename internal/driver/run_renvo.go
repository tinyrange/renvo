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
	arenaSize := backendArenaSize(target, built.Options.Tags, built.Options.ArenaSize)
	moduleLicense := built.Options.ModuleLicense
	persistMark := 0
	if resetArena {
		persistMark = arena.PersistMark()
		unit = arena.PersistBytes(unit)
		target = arena.PersistString(target)
		moduleLicense = arena.PersistString(moduleLicense)
		backendMark := mark
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

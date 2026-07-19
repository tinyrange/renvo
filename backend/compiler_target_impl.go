package main

// compileTarget composes an OS/architecture implementation after target
// selection. It is deliberately target-neutral: Linux runtime operations live
// in compiler_linux_impl.go, while target-specific image builders remain in
// their composition files until those layers are split further.
func compileTarget(input []int, output int, target int, arenaSize int) int {
	// A stage compiler is specialized while its parent is lowering this source.
	// Keep that dispatch expressed in terms of the specialization global so the
	// fixed-target branch pruner can remove every unrelated backend call.
	if compilerFixedTarget != 0 {
		if compilerFixedTarget == renvoTargetWindowsAmd64 {
			compilerFixedTarget = renvoTargetWindowsAmd64
			return compileWindowsAmd64Arena(input, output, arenaSize)
		}
		if compilerFixedTarget == renvoTargetWindows386 {
			compilerFixedTarget = renvoTargetWindows386
			return compileWindows386Arena(input, output, arenaSize)
		}
		if compilerFixedTarget == renvoTargetWindowsArm64 {
			compilerFixedTarget = renvoTargetWindowsArm64
			return compileWindowsArm64Arena(input, output, arenaSize)
		}
		if compilerFixedTarget == renvoTargetWasiWasm32 {
			compilerFixedTarget = renvoTargetWasiWasm32
			return compileWasiWasm32Arena(input, output, arenaSize)
		}
		if compilerFixedTarget == renvoTargetDarwinArm64 {
			compilerFixedTarget = renvoTargetDarwinArm64
			return compileDarwinArm64Arena(input, output, arenaSize)
		}
		if compilerFixedTarget == renvoTargetLinux386 {
			compilerFixedTarget = renvoTargetLinux386
			return compileLinux386Arena(input, output, arenaSize)
		}
		if compilerFixedTarget == renvoTargetLinuxAarch64 {
			compilerFixedTarget = renvoTargetLinuxAarch64
			return compileLinuxAarch64Arena(input, output, arenaSize)
		}
		if compilerFixedTarget == renvoTargetLinuxArm {
			compilerFixedTarget = renvoTargetLinuxArm
			return compileLinuxArmArena(input, output, arenaSize)
		}
		compilerFixedTarget = renvoTargetLinuxAmd64
		return compileLinuxAmd64Arena(input, output, arenaSize)
	}
	compilerFixedTarget = target
	if target == renvoTargetWindowsAmd64 {
		return compileWindowsAmd64Arena(input, output, arenaSize)
	}
	if target == renvoTargetWindows386 {
		return compileWindows386Arena(input, output, arenaSize)
	}
	if target == renvoTargetWindowsArm64 {
		return compileWindowsArm64Arena(input, output, arenaSize)
	}
	if target == renvoTargetWasiWasm32 {
		return compileWasiWasm32Arena(input, output, arenaSize)
	}
	if target == renvoTargetDarwinArm64 {
		return compileDarwinArm64Arena(input, output, arenaSize)
	}
	if target == renvoTargetLinux386 {
		return compileLinux386Arena(input, output, arenaSize)
	}
	if target == renvoTargetLinuxAarch64 {
		return compileLinuxAarch64Arena(input, output, arenaSize)
	}
	if target == renvoTargetLinuxArm {
		return compileLinuxArmArena(input, output, arenaSize)
	}
	if target != renvoTargetLinuxAmd64 {
		return 1
	}
	return compileLinuxAmd64Arena(input, output, arenaSize)
}

func RenvoCompileSourceToBytes(source []byte, targetName string) ([]byte, bool) {
	return RenvoCompileSourceToBytesStrip(source, targetName, false)
}

func RenvoCompileSourceToBytesStrip(source []byte, targetName string, stripSymbols bool) ([]byte, bool) {
	return RenvoCompileSourceToBytesWithOptions(source, targetName, RenvoCompileOptions{StripSymbols: stripSymbols})
}

type RenvoCompileOptions struct {
	ArenaSize    int
	StripSymbols bool
	WindowsGUI   bool
}

func RenvoDefaultArenaSize(targetName string) (int, bool) {
	target := renvoParseTargetArg(targetName)
	if target == 0 {
		return 0, false
	}
	return renvoDefaultArenaSize(target), true
}

func renvoCompileOptionsValid(target int, options RenvoCompileOptions) bool {
	if options.WindowsGUI && target != renvoTargetWindowsAmd64 && target != renvoTargetWindows386 && target != renvoTargetWindowsArm64 {
		return false
	}
	return options.ArenaSize == 0 || renvoArenaSizeValid(target, options.ArenaSize)
}

func RenvoCompileSourceToBytesWithOptions(source []byte, targetName string, options RenvoCompileOptions) ([]byte, bool) {
	target := renvoParseTargetArg(targetName)
	if target == 0 || !renvoCompileOptionsValid(target, options) {
		return nil, false
	}
	renvoSetStripSymbols(options.StripSymbols)
	renvoCompilerWindowsSubsystem = 3
	if options.WindowsGUI {
		renvoCompilerWindowsSubsystem = 2
	}
	renvoSetTarget(target)
	var prog renvoProgram
	prog = renvoParseProgram(source)
	result := renvoCompileParsedProgramArena(&prog, target, options.ArenaSize)
	if !result.ok {
		return nil, false
	}
	return result.data, true
}

func RenvoCompileSourceToOutputStrip(source []byte, targetName string, outputPath string, stripSymbols bool) bool {
	return RenvoCompileSourceToOutputWithOptions(source, targetName, outputPath, RenvoCompileOptions{StripSymbols: stripSymbols})
}

func RenvoCompileSourceToOutputWithOptions(source []byte, targetName string, outputPath string, options RenvoCompileOptions) bool {
	target := renvoParseTargetArg(targetName)
	if target == 0 || !renvoCompileOptionsValid(target, options) {
		return false
	}
	renvoSetStripSymbols(options.StripSymbols)
	renvoCompilerWindowsSubsystem = 3
	if options.WindowsGUI {
		renvoCompilerWindowsSubsystem = 2
	}
	renvoSetTarget(target)
	var prog renvoProgram
	prog = renvoParseProgram(source)
	result := renvoCompileParsedProgramArena(&prog, target, options.ArenaSize)
	if !result.ok {
		return false
	}
	output := 1
	if outputPath != "-" {
		output = open(renvoCString(outputPath), 578)
		if output < 0 {
			return false
		}
	}
	write(output, result.data, -1)
	if outputPath != "-" {
		chmod(output, 493)
		close(output)
	}
	return true
}

func RenvoCompileUnitToOutputStrip(unit []byte, targetName string, outputPath string, stripSymbols bool) bool {
	return RenvoCompileUnitToOutputStripWindowsGUI(unit, targetName, outputPath, stripSymbols, false)
}

func RenvoCompileUnitToOutputStripWindowsGUI(unit []byte, targetName string, outputPath string, stripSymbols bool, windowsGUI bool) bool {
	return RenvoCompileUnitToOutputWithOptions(unit, targetName, outputPath, RenvoCompileOptions{StripSymbols: stripSymbols, WindowsGUI: windowsGUI})
}

func RenvoCompileUnitToOutputWithOptions(unit []byte, targetName string, outputPath string, options RenvoCompileOptions) bool {
	target := renvoParseTargetArg(targetName)
	if target == 0 || !renvoCompileOptionsValid(target, options) {
		return false
	}
	renvoSetStripSymbols(options.StripSymbols)
	renvoCompilerWindowsSubsystem = 3
	if options.WindowsGUI {
		renvoCompilerWindowsSubsystem = 2
	}
	renvoSetTarget(target)
	prog, isUnit, ok := renvoDecodeUnitProgram(unit)
	if !isUnit || !ok {
		return false
	}
	result := renvoCompileParsedProgramArena(&prog, target, options.ArenaSize)
	if !result.ok {
		return false
	}
	output := 1
	if outputPath != "-" {
		output = open(renvoCString(outputPath), O_RDWR|O_CREATE|O_TRUNC)
		if output < 0 {
			return false
		}
	}
	write(output, result.data, -1)
	if outputPath != "-" {
		chmod(output, 493)
		close(output)
	}
	return true
}

func renvoCompileParsedProgram(prog *renvoProgram, target int) renvoCompileResult {
	return renvoCompileParsedProgramArena(prog, target, 0)
}

func renvoCompileParsedProgramArena(prog *renvoProgram, target int, arenaSize int) renvoCompileResult {
	var result renvoCompileResult
	if !prog.ok {
		return result
	}
	var meta renvoMeta
	renvoBuildMetaInto(prog, &meta)
	if !meta.ok {
		return result
	}
	meta.arenaSize = renvoResolveArenaSize(target, arenaSize)
	if target == renvoTargetLinux386 || target == renvoTargetWindows386 {
		return renvoTryCompileScalarProgram386(prog, &meta)
	}
	if target == renvoTargetLinuxAarch64 || target == renvoTargetDarwinArm64 || target == renvoTargetWindowsArm64 {
		return renvoTryCompileScalarProgramAarch64(prog, &meta)
	}
	if target == renvoTargetLinuxArm {
		return renvoTryCompileScalarProgramArm(prog, &meta)
	}
	if target == renvoTargetWasiWasm32 {
		return renvoTryCompileScalarProgramWasm32(prog, &meta)
	}
	return renvoTryCompileScalarProgramAmd64(prog, &meta)
}

func renvoSetStripSymbols(stripSymbols bool) {
	if stripSymbols {
		renvoCompilerStripSymbols = true
		return
	}
	renvoCompilerStripSymbols = false
}

func renvoCString(s string) string {
	var out []byte
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	out = append(out, 0)
	return string(out)
}

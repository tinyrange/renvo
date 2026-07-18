package main

// compileTarget composes an OS/architecture implementation after target
// selection. It is deliberately target-neutral: Linux runtime operations live
// in compiler_linux_impl.go, while target-specific image builders remain in
// their composition files until those layers are split further.
func compileTarget(input []int, output int, target int, arenaSize int) int {
	// A stage compiler is specialized while its parent is lowering this source.
	// Keep that dispatch expressed in terms of the specialization global so the
	// fixed-target branch pruner can remove every unrelated backend call.
	if rtgCompilerFixedTarget != 0 {
		if rtgCompilerFixedTarget == rtgTargetWindowsAmd64 {
			rtgCompilerFixedTarget = rtgTargetWindowsAmd64
			return compileWindowsAmd64Arena(input, output, arenaSize)
		}
		if rtgCompilerFixedTarget == rtgTargetWindows386 {
			rtgCompilerFixedTarget = rtgTargetWindows386
			return compileWindows386Arena(input, output, arenaSize)
		}
		if rtgCompilerFixedTarget == rtgTargetWindowsArm64 {
			rtgCompilerFixedTarget = rtgTargetWindowsArm64
			return compileWindowsArm64Arena(input, output, arenaSize)
		}
		if rtgCompilerFixedTarget == rtgTargetWasiWasm32 {
			rtgCompilerFixedTarget = rtgTargetWasiWasm32
			return compileWasiWasm32Arena(input, output, arenaSize)
		}
		if rtgCompilerFixedTarget == rtgTargetDarwinArm64 {
			rtgCompilerFixedTarget = rtgTargetDarwinArm64
			return compileDarwinArm64Arena(input, output, arenaSize)
		}
		if rtgCompilerFixedTarget == rtgTargetLinux386 {
			rtgCompilerFixedTarget = rtgTargetLinux386
			return compileLinux386Arena(input, output, arenaSize)
		}
		if rtgCompilerFixedTarget == rtgTargetLinuxAarch64 {
			rtgCompilerFixedTarget = rtgTargetLinuxAarch64
			return compileLinuxAarch64Arena(input, output, arenaSize)
		}
		if rtgCompilerFixedTarget == rtgTargetLinuxArm {
			rtgCompilerFixedTarget = rtgTargetLinuxArm
			return compileLinuxArmArena(input, output, arenaSize)
		}
		rtgCompilerFixedTarget = rtgTargetLinuxAmd64
		return compileLinuxAmd64Arena(input, output, arenaSize)
	}
	rtgCompilerFixedTarget = target
	if target == rtgTargetWindowsAmd64 {
		return compileWindowsAmd64Arena(input, output, arenaSize)
	}
	if target == rtgTargetWindows386 {
		return compileWindows386Arena(input, output, arenaSize)
	}
	if target == rtgTargetWindowsArm64 {
		return compileWindowsArm64Arena(input, output, arenaSize)
	}
	if target == rtgTargetWasiWasm32 {
		return compileWasiWasm32Arena(input, output, arenaSize)
	}
	if target == rtgTargetDarwinArm64 {
		return compileDarwinArm64Arena(input, output, arenaSize)
	}
	if target == rtgTargetLinux386 {
		return compileLinux386Arena(input, output, arenaSize)
	}
	if target == rtgTargetLinuxAarch64 {
		return compileLinuxAarch64Arena(input, output, arenaSize)
	}
	if target == rtgTargetLinuxArm {
		return compileLinuxArmArena(input, output, arenaSize)
	}
	if target != rtgTargetLinuxAmd64 {
		return 1
	}
	return compileLinuxAmd64Arena(input, output, arenaSize)
}

func RtgCompileSourceToBytes(source []byte, targetName string) ([]byte, bool) {
	return RtgCompileSourceToBytesStrip(source, targetName, false)
}

func RtgCompileSourceToBytesStrip(source []byte, targetName string, stripSymbols bool) ([]byte, bool) {
	return RtgCompileSourceToBytesWithOptions(source, targetName, RtgCompileOptions{StripSymbols: stripSymbols})
}

type RtgCompileOptions struct {
	ArenaSize    int
	StripSymbols bool
	WindowsGUI   bool
}

func RtgDefaultArenaSize(targetName string) (int, bool) {
	target := rtgParseTargetArg(targetName)
	if target == 0 {
		return 0, false
	}
	return rtgDefaultArenaSize(target), true
}

func rtgCompileOptionsValid(target int, options RtgCompileOptions) bool {
	if options.WindowsGUI && target != rtgTargetWindowsAmd64 && target != rtgTargetWindows386 && target != rtgTargetWindowsArm64 {
		return false
	}
	return options.ArenaSize == 0 || rtgArenaSizeValid(target, options.ArenaSize)
}

func RtgCompileSourceToBytesWithOptions(source []byte, targetName string, options RtgCompileOptions) ([]byte, bool) {
	target := rtgParseTargetArg(targetName)
	if target == 0 || !rtgCompileOptionsValid(target, options) {
		return nil, false
	}
	rtgSetStripSymbols(options.StripSymbols)
	rtgCompilerWindowsSubsystem = 3
	if options.WindowsGUI {
		rtgCompilerWindowsSubsystem = 2
	}
	rtgSetTarget(target)
	var prog rtgProgram
	prog = rtgParseProgram(source)
	result := rtgCompileParsedProgramArena(&prog, target, options.ArenaSize)
	if !result.ok {
		return nil, false
	}
	return result.data, true
}

func RtgCompileSourceToOutputStrip(source []byte, targetName string, outputPath string, stripSymbols bool) bool {
	return RtgCompileSourceToOutputWithOptions(source, targetName, outputPath, RtgCompileOptions{StripSymbols: stripSymbols})
}

func RtgCompileSourceToOutputWithOptions(source []byte, targetName string, outputPath string, options RtgCompileOptions) bool {
	target := rtgParseTargetArg(targetName)
	if target == 0 || !rtgCompileOptionsValid(target, options) {
		return false
	}
	rtgSetStripSymbols(options.StripSymbols)
	rtgCompilerWindowsSubsystem = 3
	if options.WindowsGUI {
		rtgCompilerWindowsSubsystem = 2
	}
	rtgSetTarget(target)
	var prog rtgProgram
	prog = rtgParseProgram(source)
	result := rtgCompileParsedProgramArena(&prog, target, options.ArenaSize)
	if !result.ok {
		return false
	}
	output := 1
	if outputPath != "-" {
		output = open(rtgCString(outputPath), 578)
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

func RtgCompileUnitToOutputStrip(unit []byte, targetName string, outputPath string, stripSymbols bool) bool {
	return RtgCompileUnitToOutputStripWindowsGUI(unit, targetName, outputPath, stripSymbols, false)
}

func RtgCompileUnitToOutputStripWindowsGUI(unit []byte, targetName string, outputPath string, stripSymbols bool, windowsGUI bool) bool {
	return RtgCompileUnitToOutputWithOptions(unit, targetName, outputPath, RtgCompileOptions{StripSymbols: stripSymbols, WindowsGUI: windowsGUI})
}

func RtgCompileUnitToOutputWithOptions(unit []byte, targetName string, outputPath string, options RtgCompileOptions) bool {
	target := rtgParseTargetArg(targetName)
	if target == 0 || !rtgCompileOptionsValid(target, options) {
		return false
	}
	rtgSetStripSymbols(options.StripSymbols)
	rtgCompilerWindowsSubsystem = 3
	if options.WindowsGUI {
		rtgCompilerWindowsSubsystem = 2
	}
	rtgSetTarget(target)
	prog, isUnit, ok := rtgDecodeUnitProgram(unit)
	if !isUnit || !ok {
		return false
	}
	result := rtgCompileParsedProgramArena(&prog, target, options.ArenaSize)
	if !result.ok {
		return false
	}
	output := 1
	if outputPath != "-" {
		output = open(rtgCString(outputPath), O_RDWR|O_CREATE|O_TRUNC)
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

func rtgCompileParsedProgram(prog *rtgProgram, target int) rtgCompileResult {
	return rtgCompileParsedProgramArena(prog, target, 0)
}

func rtgCompileParsedProgramArena(prog *rtgProgram, target int, arenaSize int) rtgCompileResult {
	var result rtgCompileResult
	if !prog.ok {
		return result
	}
	var meta rtgMeta
	rtgBuildMetaInto(prog, &meta)
	if !meta.ok {
		return result
	}
	meta.arenaSize = rtgResolveArenaSize(target, arenaSize)
	if target == rtgTargetLinux386 || target == rtgTargetWindows386 {
		return rtgTryCompileScalarProgram386(prog, &meta)
	}
	if target == rtgTargetLinuxAarch64 || target == rtgTargetDarwinArm64 || target == rtgTargetWindowsArm64 {
		return rtgTryCompileScalarProgramAarch64(prog, &meta)
	}
	if target == rtgTargetLinuxArm {
		return rtgTryCompileScalarProgramArm(prog, &meta)
	}
	if target == rtgTargetWasiWasm32 {
		return rtgTryCompileScalarProgramWasm32(prog, &meta)
	}
	return rtgTryCompileScalarProgramAmd64(prog, &meta)
}

func rtgSetStripSymbols(stripSymbols bool) {
	if stripSymbols {
		rtgCompilerStripSymbols = true
		return
	}
	rtgCompilerStripSymbols = false
}

func rtgCString(s string) string {
	var out []byte
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	out = append(out, 0)
	return string(out)
}

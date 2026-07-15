package main

// compileTarget composes an OS/architecture implementation after target
// selection. It is deliberately target-neutral: Linux runtime operations live
// in compiler_linux_impl.go, while target-specific image builders remain in
// their composition files until those layers are split further.
func compileTarget(input []int, output int, target int) int {
	// A stage compiler is specialized while its parent is lowering this source.
	// Keep that dispatch expressed in terms of the specialization global so the
	// fixed-target branch pruner can remove every unrelated backend call.
	if rtgCompilerFixedTarget != 0 {
		if rtgCompilerFixedTarget == rtgTargetWindowsAmd64 {
			rtgCompilerFixedTarget = rtgTargetWindowsAmd64
			return compileWindowsAmd64(input, output)
		}
		if rtgCompilerFixedTarget == rtgTargetWindows386 {
			rtgCompilerFixedTarget = rtgTargetWindows386
			return compileWindows386(input, output)
		}
		if rtgCompilerFixedTarget == rtgTargetWasiWasm32 {
			rtgCompilerFixedTarget = rtgTargetWasiWasm32
			return compileWasiWasm32(input, output)
		}
		if rtgCompilerFixedTarget == rtgTargetDarwinArm64 {
			rtgCompilerFixedTarget = rtgTargetDarwinArm64
			return compileDarwinArm64(input, output)
		}
		if rtgCompilerFixedTarget == rtgTargetLinux386 {
			rtgCompilerFixedTarget = rtgTargetLinux386
			return compileLinux386(input, output)
		}
		if rtgCompilerFixedTarget == rtgTargetLinuxAarch64 {
			rtgCompilerFixedTarget = rtgTargetLinuxAarch64
			return compileLinuxAarch64(input, output)
		}
		if rtgCompilerFixedTarget == rtgTargetLinuxArm {
			rtgCompilerFixedTarget = rtgTargetLinuxArm
			return compileLinuxArm(input, output)
		}
		rtgCompilerFixedTarget = rtgTargetLinuxAmd64
		return compileLinuxAmd64(input, output)
	}
	rtgCompilerFixedTarget = target
	if target == rtgTargetWindowsAmd64 {
		return compileWindowsAmd64(input, output)
	}
	if target == rtgTargetWindows386 {
		return compileWindows386(input, output)
	}
	if target == rtgTargetWasiWasm32 {
		return compileWasiWasm32(input, output)
	}
	if target == rtgTargetDarwinArm64 {
		return compileDarwinArm64(input, output)
	}
	if target == rtgTargetLinux386 {
		return compileLinux386(input, output)
	}
	if target == rtgTargetLinuxAarch64 {
		return compileLinuxAarch64(input, output)
	}
	if target == rtgTargetLinuxArm {
		return compileLinuxArm(input, output)
	}
	if target != rtgTargetLinuxAmd64 {
		return 1
	}
	return compileLinuxAmd64(input, output)
}

func RtgCompileSourceToBytes(source []byte, targetName string) ([]byte, bool) {
	return RtgCompileSourceToBytesStrip(source, targetName, false)
}

func RtgCompileSourceToBytesStrip(source []byte, targetName string, stripSymbols bool) ([]byte, bool) {
	target := rtgParseTargetArg(targetName)
	if target == 0 {
		return nil, false
	}
	rtgSetStripSymbols(stripSymbols)
	rtgCompilerWindowsSubsystem = 3
	rtgSetTarget(target)
	var prog rtgProgram
	prog = rtgParseProgram(source)
	result := rtgCompileParsedProgram(&prog, target)
	if !result.ok {
		return nil, false
	}
	return result.data, true
}

func RtgCompileSourceToOutputStrip(source []byte, targetName string, outputPath string, stripSymbols bool) bool {
	target := rtgParseTargetArg(targetName)
	if target == 0 {
		return false
	}
	rtgSetStripSymbols(stripSymbols)
	rtgCompilerWindowsSubsystem = 3
	rtgSetTarget(target)
	var prog rtgProgram
	prog = rtgParseProgram(source)
	result := rtgCompileParsedProgram(&prog, target)
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
	target := rtgParseTargetArg(targetName)
	if target == 0 || windowsGUI && target != rtgTargetWindowsAmd64 && target != rtgTargetWindows386 {
		return false
	}
	rtgSetStripSymbols(stripSymbols)
	rtgCompilerWindowsSubsystem = 3
	if windowsGUI {
		rtgCompilerWindowsSubsystem = 2
	}
	rtgSetTarget(target)
	prog, isUnit, ok := rtgDecodeUnitProgram(unit)
	if !isUnit || !ok {
		return false
	}
	result := rtgCompileParsedProgram(&prog, target)
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
	var result rtgCompileResult
	if !prog.ok {
		return result
	}
	var meta rtgMeta
	rtgBuildMetaInto(prog, &meta)
	if !meta.ok {
		return result
	}
	if target == rtgTargetLinux386 || target == rtgTargetWindows386 {
		return rtgTryCompileScalarProgram386(prog, &meta)
	}
	if target == rtgTargetLinuxAarch64 || target == rtgTargetDarwinArm64 {
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

package main

// compileTarget composes an OS/architecture implementation after target
// selection. It is deliberately target-neutral: Linux runtime operations live
// in compiler_linux_impl.go, while target-specific image builders remain in
// their composition files until those layers are split further.
func compileTarget(input []int, output int, target int, arenaSize int) int {
	// A stage compiler is specialized while its parent is lowering this source.
	// Keep that dispatch expressed in terms of the specialization global so the
	// fixed-target branch pruner can remove every unrelated backend call.
	if renvoFixedTarget != 0 {
		if renvoFixedTarget == renvoTargetWindowsAmd64 {
			renvoFixedTarget = renvoTargetWindowsAmd64
			return compileWindowsAmd64Arena(input, output, arenaSize)
		}
		if renvoFixedTarget == renvoTargetWindows386 {
			renvoFixedTarget = renvoTargetWindows386
			return compileWindows386Arena(input, output, arenaSize)
		}
		if renvoFixedTarget == renvoTargetWindowsArm64 {
			renvoFixedTarget = renvoTargetWindowsArm64
			return compileWindowsArm64Arena(input, output, arenaSize)
		}
		if renvoFixedTarget == renvoTargetWasiWasm32 {
			renvoFixedTarget = renvoTargetWasiWasm32
			return compileWasiWasm32Arena(input, output, arenaSize)
		}
		if renvoFixedTarget == renvoTargetDarwinArm64 {
			renvoFixedTarget = renvoTargetDarwinArm64
			return compileDarwinArm64Arena(input, output, arenaSize)
		}
		if renvoFixedTarget == renvoTargetLinux386 {
			renvoFixedTarget = renvoTargetLinux386
			return compileLinux386Arena(input, output, arenaSize)
		}
		if renvoFixedTarget == renvoTargetLinuxAarch64 {
			renvoFixedTarget = renvoTargetLinuxAarch64
			return compileLinuxAarch64Arena(input, output, arenaSize)
		}
		if renvoFixedTarget == renvoTargetLinuxArm {
			renvoFixedTarget = renvoTargetLinuxArm
			return compileLinuxArmArena(input, output, arenaSize)
		}
		renvoFixedTarget = renvoTargetLinuxAmd64
		return compileLinuxAmd64Arena(input, output, arenaSize)
	}
	renvoFixedTarget = target
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

// RenvoInitializeObjectCache reserves the bounded in-process object store when
// the requested target has object reuse enabled. Embedded callers invoke it
// before taking their transient frontend arena mark.
func RenvoInitializeObjectCache(targetName string) {
	target := renvoParseTargetArg(targetName)
	if target != 0 && target != renvoTargetWasiWasm32 {
		renvoInitializeObjectCache()
	}
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
	return renvoWriteCompileResult(result, outputPath)
}

func renvoWriteCompileResult(result renvoCompileResult, outputPath string) bool {
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

// RenvoCompileSession advances an embedded compilation in bounded phases. The
// Darwin/arm64 backend emits a small batch of relocatable function objects per
// step so GUI callers can return to their event loop between batches.
type RenvoCompileSession struct {
	unit       []byte
	targetName string
	outputPath string
	options    RenvoCompileOptions
	target     int
	stage      int
	done       bool
	ok         bool
	prog       *renvoProgram
	meta       *renvoMeta
	aarch64    *renvoAarch64ProgramSession
	result     renvoCompileResult
}

func RenvoBeginCompileSession(unit []byte, targetName string, outputPath string, options RenvoCompileOptions) *RenvoCompileSession {
	return &RenvoCompileSession{unit: unit, targetName: targetName, outputPath: outputPath, options: options}
}

func (s *RenvoCompileSession) Step() bool {
	if s == nil || s.done {
		return true
	}
	if s.stage == 0 {
		s.target = renvoParseTargetArg(s.targetName)
		if s.target == 0 || !renvoCompileOptionsValid(s.target, s.options) {
			s.done = true
			return true
		}
		renvoSetStripSymbols(s.options.StripSymbols)
		renvoCompilerWindowsSubsystem = 3
		if s.options.WindowsGUI {
			renvoCompilerWindowsSubsystem = 2
		}
		renvoSetTarget(s.target)
		prog, isUnit, decoded := renvoDecodeUnitProgram(s.unit)
		if !isUnit || !decoded {
			s.done = true
			return true
		}
		s.prog = &prog
		s.stage = 1
		return false
	}
	if s.stage == 1 {
		s.meta = new(renvoMeta)
		renvoBuildMetaInto(s.prog, s.meta)
		if !s.meta.ok {
			s.done = true
			return true
		}
		s.meta.arenaSize = renvoResolveArenaSize(s.target, s.options.ArenaSize)
		s.stage = 2
		return false
	}
	if s.stage == 2 {
		if s.target == renvoTargetDarwinArm64 {
			s.aarch64 = renvoBeginScalarProgramAarch64(s.prog, s.meta)
			if s.aarch64 == nil {
				s.done = true
				return true
			}
			s.stage = 3
			return false
		}
		s.result = renvoCompileProgramWithMeta(s.prog, s.meta, s.target)
		s.stage = 4
		return false
	}
	if s.stage == 3 {
		if !s.aarch64.step(8) {
			return false
		}
		s.result = s.aarch64.result
		s.stage = 4
		return false
	}
	s.ok = renvoWriteCompileResult(s.result, s.outputPath)
	s.done = true
	return true
}

func (s *RenvoCompileSession) Result() bool {
	return s != nil && s.done && s.ok
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
	return renvoCompileProgramWithMetaScratch(prog, &meta, target)
}

func renvoCompileProgramWithMetaScratch(prog *renvoProgram, meta *renvoMeta, target int) renvoCompileResult {
	if target == renvoTargetLinux386 || target == renvoTargetWindows386 {
		return renvoTryCompileScalarProgram386Scratch(prog, meta)
	}
	if target == renvoTargetLinuxAarch64 || target == renvoTargetDarwinArm64 || target == renvoTargetWindowsArm64 {
		return renvoTryCompileScalarProgramAarch64Scratch(prog, meta)
	}
	if target == renvoTargetLinuxArm {
		return renvoTryCompileScalarProgramArmScratch(prog, meta)
	}
	if target == renvoTargetWasiWasm32 {
		return renvoTryCompileScalarProgramWasm32(prog, meta)
	}
	return renvoTryCompileScalarProgramAmd64Scratch(prog, meta)
}

func renvoCompileProgramWithMeta(prog *renvoProgram, meta *renvoMeta, target int) renvoCompileResult {
	if target == renvoTargetLinux386 || target == renvoTargetWindows386 {
		return renvoTryCompileScalarProgram386Cached(prog, meta)
	}
	if target == renvoTargetLinuxAarch64 || target == renvoTargetDarwinArm64 || target == renvoTargetWindowsArm64 {
		return renvoTryCompileScalarProgramAarch64Cached(prog, meta)
	}
	if target == renvoTargetLinuxArm {
		return renvoTryCompileScalarProgramArmCached(prog, meta)
	}
	if target == renvoTargetWasiWasm32 {
		return renvoTryCompileScalarProgramWasm32(prog, meta)
	}
	return renvoTryCompileScalarProgramAmd64Cached(prog, meta)
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

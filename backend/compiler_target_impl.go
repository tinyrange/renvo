package main

func renvoStructArgByReference(g *renvoLinearGen, kind int) bool {
	return kind == renvoTypeStruct && g.c.renvoTargetArch != renvoArchWasm32 &&
		(g.c.renvoNativeIntSize == 4 || renvoPreparedBackendActive != 0 && g.c.renvoNativeIntSize == 2)
}

func renvoRTGEnsureStringEqualHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	a := &g.asm
	if g.streqEmitted {
		return g.streqLabel
	}
	g.streqEmitted = true
	g.streqLabel = renvoAsmNewLabel(a)
	if renvoRTGStructuredFunctions != 0 {
		renvoQueueStructuredHelper(g, renvoStructuredHelperStringEqual, 0, g.streqLabel)
		return g.streqLabel
	}
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.streqLabel)
	renvoRTGEmitStringEqualHelperBody(g)
	renvoAsmMarkLabel(a, afterLabel)
	return g.streqLabel
}

func renvoRTGEmitStringEqualHelperBody(g *renvoLinearGen) {
	a := &g.asm
	notEqualLabel := renvoAsmNewLabel(a)
	equalLabel := renvoAsmNewLabel(a)
	loopLabel := renvoAsmNewLabel(a)

	// String equality receives (left data, left length, right data, right
	// length) in the first four ABI call words and returns a boolean in primary.
	renvoRTGDirectCompare(a, renvoRTGCallWord1, renvoRTGCallWord3)
	renvoAsmJnzLabel(a, notEqualLabel)
	renvoRTGDirectMoveImmediate(a, renvoRTGCallWord4, 0)
	renvoRTGDirectCompare(a, renvoRTGCallWord1, renvoRTGCallWord4)
	renvoAsmJzLabel(a, equalLabel)
	renvoAsmMarkLabel(a, loopLabel)
	renvoRTGDirectLoadU8(a, renvoRTGScratch,
		renvoRTGAsmAddress(renvoRTGCallWord0, RTGNoRegister, 0, 1))
	renvoRTGDirectLoadU8(a, renvoRTGCallWord4,
		renvoRTGAsmAddress(renvoRTGCallWord2, RTGNoRegister, 0, 1))
	renvoRTGDirectCompare(a, renvoRTGScratch, renvoRTGCallWord4)
	renvoAsmJnzLabel(a, notEqualLabel)
	renvoRTGDirectIncrement(a, renvoRTGCallWord0)
	renvoRTGDirectIncrement(a, renvoRTGCallWord2)
	renvoRTGDirectDecrement(a, renvoRTGCallWord1)
	renvoRTGDirectMoveImmediate(a, renvoRTGCallWord4, 0)
	renvoRTGDirectCompare(a, renvoRTGCallWord1, renvoRTGCallWord4)
	renvoAsmJnzLabel(a, loopLabel)
	renvoAsmMarkLabel(a, equalLabel)
	renvoRTGDirectMoveImmediate(a, renvoRTGPrimary, 1)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, notEqualLabel)
	renvoRTGDirectMoveImmediate(a, renvoRTGPrimary, 0)
	renvoAsmRet(a)
}

// compileTarget composes an OS/architecture implementation after target
// selection. It is deliberately target-neutral: Linux runtime operations live
// in compiler_linux_impl.go, while target-specific image builders remain in
// their composition files until those layers are split further.
func compileTarget(input []int, output int, target int, arenaSize int) int {
	if renvoPreparedBackendActive != 0 || renvoFixedTarget == 0 && target == renvoTargetRTG {
		// renvoCompileUnitInput uses a positional header read for regular files,
		// so a non-unit input is still positioned at its first byte here. Prepared
		// backends do not carry the architecture-specific compile*Arena wrappers;
		// parse their single raw source input through the shared RTG path instead.
		if len(input) != 1 {
			renvoPrintErr("renvo: prepared backends require one input file\n")
			return 1
		}
		var src []byte
		src = renvoReadAll(input[0], src)
		prog := renvoParseProgram(src)
		return renvoCompileProgramToOutput(&prog, output, target, arenaSize)
	}
	// A stage compiler is specialized while its parent is lowering this source.
	// Keep that dispatch expressed in terms of the specialization global so the
	// fixed-target branch pruner can remove every unrelated backend call.
	if renvoFixedTarget != 0 {
		if renvoFixedTarget == renvoTargetLinuxKernelAmd64 {
			renvoFixedTarget = renvoTargetLinuxKernelAmd64
			return compileLinuxAmd64Arena(input, output, arenaSize)
		}
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
		if renvoFixedTarget == renvoTargetVM32 {
			renvoFixedTarget = renvoTargetVM32
			return compileVM32Arena(input, output, arenaSize)
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
		if renvoFixedTarget >= renvoTargetFreeBSDAmd64 && renvoFixedTarget <= renvoTargetNetBSDAmd64 {
			return compileBSDAmd64Arena(input, output, renvoFixedTarget, arenaSize)
		}
		renvoFixedTarget = renvoTargetLinuxAmd64
		return compileLinuxAmd64Arena(input, output, arenaSize)
	}
	if target == renvoTargetLinuxKernelAmd64 {
		return compileLinuxKernelAmd64Arena(input, output, arenaSize)
	}
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
	if target == renvoTargetWasiWasm32 || target == renvoTargetVM32 {
		return compileVM32Arena(input, output, arenaSize)
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
	if target >= renvoTargetFreeBSDAmd64 && target <= renvoTargetNetBSDAmd64 {
		return compileBSDAmd64Arena(input, output, target, arenaSize)
	}
	if target != renvoTargetLinuxAmd64 {
		return 1
	}
	return compileLinuxAmd64Arena(input, output, arenaSize)
}

func compileBSDAmd64Arena(input []int, output int, target int, arenaSize int) int {
	renvoSetTarget(target)
	return renvoCompileAmd64(input, output, arenaSize)
}

func RenvoCompileSourceToBytes(source []byte, targetName string) ([]byte, bool) {
	return RenvoCompileSourceToBytesStrip(source, targetName, false)
}

func RenvoCompileSourceToBytesStrip(source []byte, targetName string, stripSymbols bool) ([]byte, bool) {
	return RenvoCompileSourceToBytesWithOptions(source, targetName, RenvoCompileOptions{StripSymbols: stripSymbols})
}

type RenvoCompileOptions struct {
	ArenaSize      int
	StripSymbols   bool
	WindowsGUI     bool
	EmitImage      bool
	ModuleLicense  string
	ModuleNamePath string
	ObjectFile     bool
	Code16         bool
	RegParm        int
}

// RenvoInitializeObjectCache reserves the bounded in-process object store when
// the requested target has object reuse enabled. Embedded callers invoke it
// before taking their transient frontend arena mark.
func RenvoInitializeObjectCache(targetName string) {
	target := renvoParseTargetArg(targetName)
	if target != 0 && target != renvoTargetWasiWasm32 && target != renvoTargetVM32 && target != renvoTargetLinuxKernelAmd64 {
		renvoInitializeObjectCache()
	}
}

func RenvoTargetSupported(targetName string) bool {
	return renvoParseTargetArg(targetName) != 0
}

// RenvoTargetBinding returns the descriptor identity used to bind frontend
// units to a target. Prepared compilers use this to advertise their embedded
// target to the frontend as well as to the backend dispatcher.
func RenvoTargetBinding(targetName string) (string, string, int, bool) {
	target := renvoParseTargetArg(targetName)
	if target == 0 {
		return "", "", 0, false
	}
	return renvoRTGTargetBinding(target)
}

// RenvoTargetHasBuildTag reports the source-selection tags exported by a
// prepared target descriptor.
func RenvoTargetHasBuildTag(targetName string, tag string) bool {
	target := renvoParseTargetArg(targetName)
	return target != 0 && renvoRTGTargetHasBuildTag(target, tag)
}

// RenvoTargetHasCapability reports one capability from the selected target
// descriptor. Multi-target frontends use it to distinguish complete artifacts
// from images whose entrypoint can execute directly at their embedded address.
func RenvoTargetHasCapability(targetName string, capability string) bool {
	target := renvoParseTargetArg(targetName)
	return target != 0 && renvoRTGTargetHasCapability(target, capability)
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
	if options.Code16 && (target != renvoTargetLinux386 || !options.ObjectFile) {
		return false
	}
	if options.RegParm != 0 && (options.RegParm != 3 || target != renvoTargetLinux386 || !options.ObjectFile) {
		return false
	}
	return options.ArenaSize == 0 || options.ArenaSize >= renvoArenaSizeMinimum && options.ArenaSize <= renvoArenaSizeMaximum
}

func RenvoCompileSourceToBytesWithOptions(source []byte, targetName string, options RenvoCompileOptions) ([]byte, bool) {
	target := renvoParseTargetArg(targetName)
	if target == 0 || !renvoCompileOptionsValid(target, options) {
		return nil, false
	}
	context := renvoNewCompileContext(target, options.StripSymbols, options.WindowsGUI, options.EmitImage)
	context.objectFile = options.ObjectFile
	context.code16 = options.Code16
	context.regParm = options.RegParm
	moduleNamePath := options.ModuleNamePath
	if moduleNamePath == "" {
		moduleNamePath = "renvo"
	}
	renvoConfigureCompileContext(context, targetName, moduleNamePath, options.ModuleLicense)
	prog := renvoParseProgramWithContext(source, context)
	if prog.ok && target == renvoTargetVM32 && renvoProgramNeedsSoftFloat(&prog) {
		source = renvoAppendSoftFloatSource(source)
		prog = renvoParseProgramWithContext(source, context)
	}
	result := renvoCompileParsedProgramArena(&prog, target, options.ArenaSize)
	if !result.ok {
		return nil, false
	}
	return renvoCompileOutputDataWithContext(context, result.data, target), true
}

func RenvoCompileSourceToOutputStrip(source []byte, targetName string, outputPath string, stripSymbols bool) bool {
	return RenvoCompileSourceToOutputWithOptions(source, targetName, outputPath, RenvoCompileOptions{StripSymbols: stripSymbols})
}

func RenvoCompileSourceToOutputWithOptions(source []byte, targetName string, outputPath string, options RenvoCompileOptions) bool {
	target := renvoParseTargetArg(targetName)
	if target == 0 || !renvoCompileOptionsValid(target, options) {
		return false
	}
	context := renvoNewCompileContext(target, options.StripSymbols, options.WindowsGUI, options.EmitImage)
	context.objectFile = options.ObjectFile
	context.code16 = options.Code16
	context.regParm = options.RegParm
	renvoConfigureCompileContext(context, targetName, outputPath, options.ModuleLicense)
	prog := renvoParseProgramWithContext(source, context)
	if prog.ok && target == renvoTargetVM32 && renvoProgramNeedsSoftFloat(&prog) {
		source = renvoAppendSoftFloatSource(source)
		prog = renvoParseProgramWithContext(source, context)
	}
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
	write(output, renvoCompileOutputDataWithContext(context, result.data, target), -1)
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
	if target == 0 {
		renvoPrintErr("renvo: backend rejected unknown target\n")
		return false
	}
	if !renvoCompileOptionsValid(target, options) {
		renvoPrintErr("renvo: backend rejected compile options\n")
		return false
	}
	if !renvoUnitBindingMatchesTarget(unit, target) {
		renvoPrintErr("renvo: frontend unit target binding does not match backend\n")
		return false
	}
	context := renvoNewCompileContext(target, options.StripSymbols, options.WindowsGUI, options.EmitImage)
	context.objectFile = options.ObjectFile
	context.code16 = options.Code16
	context.regParm = options.RegParm
	renvoConfigureCompileContext(context, targetName, outputPath, options.ModuleLicense)
	prog, isUnit, ok := renvoDecodeUnitProgram(unit)
	if !isUnit || !ok {
		renvoPrintErr("renvo: backend could not decode frontend unit\n")
		return false
	}
	prog.c = *context
	result := renvoCompileParsedProgramArena(&prog, target, options.ArenaSize)
	return renvoWriteCompileResult(context, result, outputPath)
}

// RenvoCompileUnitToBytesWithOptions exposes the same linked result without
// routing it through a filesystem descriptor. The bundled frontend uses this
// path for script execution so the RNVI transport can remain in memory.
func RenvoCompileUnitToBytesWithOptions(unit []byte, targetName string, options RenvoCompileOptions) ([]byte, bool) {
	target := renvoParseTargetArg(targetName)
	if target == 0 || !renvoCompileOptionsValid(target, options) ||
		!renvoUnitBindingMatchesTarget(unit, target) {
		return nil, false
	}
	context := renvoNewCompileContext(target, options.StripSymbols, options.WindowsGUI, options.EmitImage)
	context.objectFile = options.ObjectFile
	context.code16 = options.Code16
	context.regParm = options.RegParm
	moduleNamePath := options.ModuleNamePath
	if moduleNamePath == "" {
		moduleNamePath = "renvo"
	}
	renvoConfigureCompileContext(context, targetName, moduleNamePath, options.ModuleLicense)
	prog, isUnit, ok := renvoDecodeUnitProgram(unit)
	if !isUnit || !ok {
		return nil, false
	}
	prog.c = *context
	result := renvoCompileParsedProgramArena(&prog, target, options.ArenaSize)
	if !result.ok {
		return nil, false
	}
	return renvoCompileOutputDataWithContext(context, result.data, target), true
}

func renvoWriteCompileResult(context *renvoCompileContext, result renvoCompileResult, outputPath string) bool {
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
	write(output, renvoCompileOutputDataWithContext(context, result.data, context.renvoTarget), -1)
	if outputPath != "-" {
		mode := 493
		if context.objectFile {
			mode = 420
		}
		chmod(output, mode)
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
	context    *renvoCompileContext
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
		if s.target == 0 || !renvoCompileOptionsValid(s.target, s.options) ||
			!renvoUnitBindingMatchesTarget(s.unit, s.target) {
			s.done = true
			return true
		}
		s.context = renvoNewCompileContext(s.target, s.options.StripSymbols, s.options.WindowsGUI, s.options.EmitImage)
		s.context.objectFile = s.options.ObjectFile
		s.context.code16 = s.options.Code16
		s.context.regParm = s.options.RegParm
		renvoConfigureCompileContext(s.context, s.targetName, s.outputPath, s.options.ModuleLicense)
		prog, isUnit, decoded := renvoDecodeUnitProgram(s.unit)
		if !isUnit || !decoded {
			s.done = true
			return true
		}
		prog.c = *s.context
		s.prog = &prog
		s.stage = 1
		return false
	}
	if s.stage == 1 {
		if s.target == renvoTargetLinuxKernelAmd64 && !s.context.objectFile {
			if !renvoPrepareKernelMetadata() {
				s.done = true
				return true
			}
			renvoCaptureKernelCompileContext(s.context)
			s.prog.c = *s.context
		}
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
	s.ok = renvoWriteCompileResult(s.context, s.result, s.outputPath)
	s.done = true
	return true
}

func (s *RenvoCompileSession) Result() bool {
	return s != nil && s.done && s.ok
}

func renvoCompileParsedProgram(prog *renvoProgram, target int) renvoCompileResult {
	if prog.c.renvoTarget == 0 {
		prog.c = *renvoLegacyCompileContext()
	}
	return renvoCompileParsedProgramArena(prog, target, 0)
}

func renvoCompileParsedProgramArena(prog *renvoProgram, target int, arenaSize int) renvoCompileResult {
	var result renvoCompileResult
	if !prog.ok {
		return result
	}
	if !prog.c.objectFile && (target == renvoTargetLinuxKernelAmd64 || target == renvoTargetRTG &&
		renvoRTGPreparedKernelModule != 0) {
		if !renvoPrepareKernelMetadata() {
			return result
		}
		if target == renvoTargetLinuxKernelAmd64 {
			renvoCaptureKernelCompileContext(&prog.c)
		} else {
			renvoPopulateKernelCompileContext(&prog.c)
		}
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
	if renvoPreparedBackendActive != 0 || renvoFixedTarget == 0 && target == renvoTargetRTG {
		return renvoTryCompileScalarProgramRTG(prog, meta)
	}
	if target == renvoTargetLinux386 || target == renvoTargetWindows386 {
		return renvoTryCompileScalarProgram386Scratch(prog, meta)
	}
	if target == renvoTargetLinuxAarch64 || target == renvoTargetDarwinArm64 || target == renvoTargetWindowsArm64 {
		return renvoTryCompileScalarProgramAarch64Scratch(prog, meta)
	}
	if target == renvoTargetLinuxArm {
		return renvoTryCompileScalarProgramArmScratch(prog, meta)
	}
	if target == renvoTargetWasiWasm32 || target == renvoTargetVM32 {
		return renvoTryCompileScalarProgramWasm32(prog, meta)
	}
	return renvoTryCompileScalarProgramAmd64Scratch(prog, meta)
}

func renvoCompileProgramWithMeta(prog *renvoProgram, meta *renvoMeta, target int) renvoCompileResult {
	if renvoPreparedBackendActive != 0 || renvoFixedTarget == 0 && target == renvoTargetRTG {
		return renvoTryCompileScalarProgramRTG(prog, meta)
	}
	if target == renvoTargetLinuxKernelAmd64 {
		return renvoTryCompileScalarProgramAmd64Scratch(prog, meta)
	}
	if target == renvoTargetLinux386 || target == renvoTargetWindows386 {
		return renvoTryCompileScalarProgram386Cached(prog, meta)
	}
	if target == renvoTargetLinuxAarch64 || target == renvoTargetDarwinArm64 || target == renvoTargetWindowsArm64 {
		return renvoTryCompileScalarProgramAarch64Cached(prog, meta)
	}
	if target == renvoTargetLinuxArm {
		return renvoTryCompileScalarProgramArmCached(prog, meta)
	}
	if target == renvoTargetWasiWasm32 || target == renvoTargetVM32 {
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

func renvoConfigureTargetMode(targetName string, outputPath string) {
	renvoKernelRelease = ""
	renvoKernelBTF = nil
	renvoKernelSymvers = nil
	renvoKernelVersion = ""
	renvoKernelModuleSize = 0
	renvoKernelModuleNameOff = -1
	renvoKernelModuleInitOff = -1
	renvoKernelModuleExitOff = -1
	renvoKernelModuleName = renvoKernelNameFromOutput(outputPath)
	renvoKernelLicense = "Proprietary"
}

func renvoConfigureCompileContext(context *renvoCompileContext, targetName string, outputPath string, moduleLicense string) {
	renvoNonNil(context)
	if !targetIsKernelModule(context) {
		return
	}
	context.kernel = new(renvoKernelCompileContext)
	kernel := context.kernel
	kernel.kernelNameOff = -1
	kernel.kernelInitOff = -1
	kernel.kernelExitOff = -1
	kernel.kernelModuleName = renvoKernelNameFromOutput(outputPath)
	kernel.kernelLicense = "Proprietary"
	if moduleLicense != "" {
		kernel.kernelLicense = moduleLicense
	}
}

func renvoCaptureKernelCompileContext(context *renvoCompileContext) {
	renvoNonNil(context)
	context.renvoTarget = renvoTargetLinuxKernelAmd64
	context.renvoTargetOS = renvoOSLinux
	context.renvoTargetArch = renvoArchAmd64
	context.renvoNativeIntSize = 8
	renvoPopulateKernelCompileContext(context)
}

func renvoPopulateKernelCompileContext(context *renvoCompileContext) {
	renvoNonNil(context)
	if context.kernel == nil {
		context.kernel = new(renvoKernelCompileContext)
		context.kernel.kernelModuleName = renvoKernelModuleName
		context.kernel.kernelLicense = renvoKernelLicense
	}
	context.kernel.kernelModuleSize = renvoKernelModuleSize
	context.kernel.kernelNameOff = renvoKernelModuleNameOff
	context.kernel.kernelInitOff = renvoKernelModuleInitOff
	context.kernel.kernelExitOff = renvoKernelModuleExitOff
	context.kernel.kernelRelease = renvoKernelRelease
	context.kernel.kernelVersion = renvoKernelVersion
	context.kernel.kernelBTF = renvoKernelBTF
	context.kernel.kernelSymvers = renvoKernelSymvers
}

func renvoSetKernelLicense(license string) {
	if license != "" {
		renvoKernelLicense = license
	}
}

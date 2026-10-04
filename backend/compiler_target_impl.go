package main

// renvoStoreTargetConstant keeps target object byte order out of source-level
// constant evaluation. The profile is descriptor-derived for prepared targets.
func renvoStoreTargetConstant(c *renvoCompileContext, data []byte, offset int, size int, bits uint64) bool {
	target := c.renvoTarget
	if renvoFixedTarget != 0 {
		target = renvoFixedTarget
	}
	endian := 0
	if target == renvoTargetRTG {
		profile := renvoRTGProfileForTarget(target)
		endian = profile.endian
	} else if target > 0 && target < len(renvoTargetEndianTable) {
		endian = int(renvoTargetEndianTable[target])
	}
	return renvoStoreIntegerBytes(data, offset, size, bits, endian)
}

func renvoStoreIntegerBytes(data []byte, offset int, size int, bits uint64, endian int) bool {
	if size < 1 || size > 8 || offset < 0 || offset > len(data) || size > len(data)-offset ||
		endian != renvoEndianLittle && endian != renvoEndianBig {
		return false
	}
	for at := 0; at < size; at++ {
		shift := at
		if endian == renvoEndianBig {
			shift = size - 1 - at
		}
		data[offset+at] = byte(bits >> (shift * 8))
	}
	return true
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

// compileTarget validates target selection before entering the shared source
// pipeline. Prepared adapters retain their single-input contract.
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
	if renvoFixedTarget != 0 {
		target = renvoFixedTarget
	}
	if target <= 0 || target >= len(targetArchTable) {
		return 1
	}
	renvoSetTarget(target)
	return renvoCompileSourceInputs(input, output, arenaSize)
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
	if target != 0 && renvoProgramCacheSupported(renvoNewCompileContext(target, false, false, false)) {
		renvoInitializeObjectCache()
	}
}

func RenvoTargetSupported(targetName string) bool {
	return renvoParseTargetArg(targetName) != 0
}

// RenvoTargetLayout exposes the destination widths to the frontend of a
// prepared compiler, including targets absent from the built-in catalog.
func RenvoTargetLayout(targetName string) (int, int, bool) {
	target := renvoParseTargetArg(targetName)
	if target == renvoTargetRTG {
		profile := renvoRTGProfileForTarget(target)
		return profile.intBits, profile.pointerBits, profile.intBits != 0
	}
	profile, ok := renvoProfileForTarget(target)
	return profile.intBits, profile.pointerBits, ok
}

func RenvoTargetScalarAlignment(targetName string) (int, bool) {
	target := renvoParseTargetArg(targetName)
	profile, ok := renvoProfileForTarget(target)
	if target == renvoTargetRTG {
		profile = renvoRTGProfileForTarget(target)
		ok = profile.intBits != 0
	}
	context := renvoCompileContext{renvoNativeIntSize: profile.intBits / 8, renvoTargetArch: profile.arch}
	return renvoNativeAlignment(&context, 8), ok
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
// cache-capable backends emit a small batch of relocatable function objects per
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
	program    *renvoProgramSession
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
		if renvoKernelProgram(s.context) && !s.context.objectFile {
			if !renvoPrepareKernelMetadata(s.context) {
				s.done = true
				return true
			}
			renvoPopulateKernelCompileContext(s.context)
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
		if renvoProgramCacheSupported(s.context) {
			s.program = renvoBeginProgramSession(s.prog, s.meta)
			if s.program == nil {
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
		if !s.program.step(8) {
			return false
		}
		s.result = s.program.result
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
	if renvoKernelProgram(&prog.c) && !prog.c.objectFile {
		if !renvoPrepareKernelMetadata(&prog.c) {
			return result
		}
		renvoPopulateKernelCompileContext(&prog.c)
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
	return renvoTryCompileScalarProgramScratch(prog, meta)
}

func renvoCompileProgramWithMeta(prog *renvoProgram, meta *renvoMeta, target int) renvoCompileResult {
	if !renvoProgramCacheSupported(meta.c) {
		return renvoTryCompileScalarProgramScratch(prog, meta)
	}
	return renvoTryCompileScalarProgramCached(prog, meta)
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
	if len(context.kernel.kernelBTF) != 0 && len(context.kernel.kernelSymvers) != 0 {
		return
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

// RenvoEmitPureBlock preserves the public two-native-target RFE adapter. The
// record validation and lowering are target-neutral and take an explicit context.
func RenvoEmitPureBlock(records []int, stateWords int, arm64 bool) ([]byte, bool) {
	arch := renvoArchAmd64
	if arm64 {
		arch = renvoArchAarch64
	}
	// Do not allocate the whole-program emitter's multi-megabyte reserves for a
	// small block. No global compiler options or legacy context are consulted.
	context := &renvoCompileContext{renvoTargetArch: arch, renvoTargetOS: renvoOSLinux, renvoNativeIntSize: 8, stripSymbols: true}
	return renvoEmitPureBlock(records, stateWords, context)
}

// renvoParseProgram adapts the legacy global target selection to the shared
// parser. Explicit-context callers use renvoParseProgramWithContext instead.
func renvoParseProgram(src []byte) renvoProgram {
	var p renvoProgram
	p.c.stripSymbols = renvoCompilerStripSymbols
	if renvoFixedTarget == 0 {
		p.c.renvoTarget = renvoTarget
		p.c.renvoTargetOS = renvoTargetOS
		p.c.renvoTargetArch = renvoTargetArch
		p.c.renvoNativeIntSize = renvoNativeIntSize
		p.c.windowsSubsystem = renvoCompilerWindowsSubsystem
		p.c.emitImage = renvoCompilerEmitImage
	} else if targetIsWindows(renvoTargetOS) {
		p.c.windowsSubsystem = renvoCompilerWindowsSubsystem
	}
	renvoParseProgramInto(src, &p)
	return p
}

// Compatibility classification for definition-owned object policies. Shared
// language lowering queries individual capabilities rather than these ABI IDs.
const renvoObjectABIUnavailable = 0
const renvoObjectABISysV = 1
const renvoObjectABICdecl = 2

func renvoIsSysVObject(c *renvoCompileContext) bool {
	return c != nil && c.objectFile && renvoTargetObjectCallABI(c) == renvoObjectABISysV
}

func renvoIsCdeclObject(c *renvoCompileContext) bool {
	return c != nil && c.objectFile && renvoTargetObjectCallABI(c) == renvoObjectABICdecl
}

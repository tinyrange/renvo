package main

var renvoDefaultTarget int = renvoTargetLinuxAmd64
var renvoFixedTarget int
var renvoCompilerStripSymbols bool

func renvoOpenArg(path string, env []string) int {
	directFd := open(path, O_RDONLY)
	if directFd >= 0 {
		return directFd
	}
	for e := 0; e < len(env); e++ {
		pwd := env[e]
		if pwd[0] == 'P' && pwd[1] == 'W' && pwd[2] == 'D' && pwd[3] == '=' {
			var full []byte
			for i := 4; i < len(pwd); i++ {
				full = append(full, pwd[i])
			}
			full = append(full, '/')
			for i := 0; i < len(path); i++ {
				full = append(full, path[i])
			}
			fd := open(string(full), O_RDONLY)
			if fd >= 0 {
				return fd
			}
			full = append(full, 0)
			return open(string(full), O_RDONLY)
		}
	}
	return -1
}

func renvoParseTargetArg(target string) int {
	if len(target) == 11 && target[0] == 'l' && target[1] == 'i' && target[2] == 'n' && target[3] == 'u' && target[4] == 'x' && target[5] == '/' && target[6] == 'a' && target[7] == 'm' && target[8] == 'd' && target[9] == '6' && target[10] == '4' {
		return renvoTargetLinuxAmd64
	}
	if len(target) == 9 && target[0] == 'l' && target[1] == 'i' && target[2] == 'n' && target[3] == 'u' && target[4] == 'x' && target[5] == '/' && target[6] == '3' && target[7] == '8' && target[8] == '6' {
		return renvoTargetLinux386
	}
	if len(target) == 13 && target[0] == 'l' && target[1] == 'i' && target[2] == 'n' && target[3] == 'u' && target[4] == 'x' && target[5] == '/' && target[6] == 'a' && target[7] == 'a' && target[8] == 'r' && target[9] == 'c' && target[10] == 'h' && target[11] == '6' && target[12] == '4' {
		return renvoTargetLinuxAarch64
	}
	if len(target) == 9 && target[0] == 'l' && target[1] == 'i' && target[2] == 'n' && target[3] == 'u' && target[4] == 'x' && target[5] == '/' && target[6] == 'a' && target[7] == 'r' && target[8] == 'm' {
		return renvoTargetLinuxArm
	}
	if len(target) == 13 && target[0] == 'w' && target[1] == 'i' && target[2] == 'n' && target[3] == 'd' && target[4] == 'o' && target[5] == 'w' && target[6] == 's' && target[7] == '/' && target[8] == 'a' && target[9] == 'm' && target[10] == 'd' && target[11] == '6' && target[12] == '4' {
		return renvoTargetWindowsAmd64
	}
	if len(target) == 11 && target[0] == 'w' && target[1] == 'i' && target[2] == 'n' && target[3] == 'd' && target[4] == 'o' && target[5] == 'w' && target[6] == 's' && target[7] == '/' && target[8] == '3' && target[9] == '8' && target[10] == '6' {
		return renvoTargetWindows386
	}
	if len(target) == 13 && target[0] == 'w' && target[1] == 'i' && target[2] == 'n' && target[3] == 'd' && target[4] == 'o' && target[5] == 'w' && target[6] == 's' && target[7] == '/' && target[8] == 'a' && target[9] == 'r' && target[10] == 'm' && target[11] == '6' && target[12] == '4' {
		return renvoTargetWindowsArm64
	}
	if len(target) == 11 && target[0] == 'w' && target[1] == 'a' && target[2] == 's' && target[3] == 'i' && target[4] == '/' && target[5] == 'w' && target[6] == 'a' && target[7] == 's' && target[8] == 'm' && target[9] == '3' && target[10] == '2' {
		return renvoTargetWasiWasm32
	}
	if len(target) == 12 && target[0] == 'd' && target[1] == 'a' && target[2] == 'r' && target[3] == 'w' && target[4] == 'i' && target[5] == 'n' && target[6] == '/' && target[7] == 'a' && target[8] == 'r' && target[9] == 'm' && target[10] == '6' && target[11] == '4' {
		return renvoTargetDarwinArm64
	}
	return 0
}

func renvoPrintErr(s string) {
	write(2, []byte(s), -1)
}

func renvoPrintIntErr(v int) {
	if v == 0 {
		renvoPrintErr("0")
		return
	}
	if v < 0 {
		renvoPrintErr("-")
		v = -v
	}
	var digits []byte
	for v > 0 {
		digits = append(digits, byte('0'+v%10))
		v = v / 10
	}
	for i := len(digits) - 1; i >= 0; i-- {
		write(2, digits[i:i+1], -1)
	}
}

func renvoPrintUsage() {
	renvoPrintErr("usage: renvo [options] -o <output|-> <input.go|->...\n")
}

func renvoParsePositiveDecimal(value string) (int, bool) {
	if len(value) == 0 {
		return 0, false
	}
	result := 0
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch < '0' || ch > '9' {
			return 0, false
		}
		digit := int(ch - '0')
		if result > (1073741824-digit)/10 {
			return 0, false
		}
		result = result*10 + digit
	}
	if result < 256 || result > 1073741824 {
		return 0, false
	}
	return result, true
}

func renvoPrintUnsupportedTarget(target string) {
	renvoPrintErr("renvo: unsupported target: ")
	renvoPrintErr(target)
	renvoPrintErr("\n")
	renvoPrintErr("renvo: supported targets: linux/amd64, linux/386, linux/aarch64, linux/arm, windows/amd64, windows/386, windows/arm64, wasi/wasm32, darwin/arm64\n")
}

func renvoUnitRead32(src []byte, pos int) int {
	return int(src[pos]) | (int(src[pos+1]) << 8) | (int(src[pos+2]) << 16) | (int(src[pos+3]) << 24)
}

type renvoUnitReader struct {
	src []byte
	pos int
	end int
	ok  bool
}

func renvoUnitReadVar(r *renvoUnitReader) int {
	renvoNonNil(r)
	if !r.ok || r.pos >= r.end {
		r.ok = false
		return 0
	}
	first := r.src[r.pos]
	r.pos++
	if first < 0x80 {
		return int(first)
	}
	value := int(first & 0x7f)
	shift := 7
	for r.pos < r.end && shift <= 28 {
		b := r.src[r.pos]
		r.pos++
		if shift >= 28 && b >= 0x10 {
			r.ok = false
			return 0
		}
		value = value | (int(b&0x7f) << shift)
		if b < 0x80 {
			if shift > 0 && b == 0 {
				r.ok = false
				return 0
			}
			return value
		}
		shift = shift + 7
	}
	r.ok = false
	return 0
}

func renvoDecodeUnitTokens(text []byte, data []byte) ([]int32, bool) {
	r := renvoUnitReader{src: data, end: len(data), ok: true}
	count := renvoUnitReadVar(&r)
	if !r.ok {
		return nil, false
	}
	out := make([]int32, 0, count*renvoTokenStride)
	start := 0
	line := 0
	for i := 0; i < count; i++ {
		kind := renvoUnitReadVar(&r)
		delta := renvoUnitReadVar(&r)
		size := renvoUnitReadVar(&r)
		lineDelta := renvoUnitReadVar(&r)
		if !r.ok {
			return nil, false
		}
		start = start + delta
		line = line + lineDelta
		if kind < 0 || kind > 255 || start < 0 || start > 0xffffff || size < 0 || line < 0 || line > 0xffff || start+size > len(text) {
			return nil, false
		}
		if kind == renvoTokOp {
			if size > 255 {
				return nil, false
			}
		} else if size > 0xffff {
			return nil, false
		}
		base := len(out)
		out = out[:base+renvoTokenStride]
		charBits := 0
		if kind == renvoTokOp && size == 1 {
			charBits = int(text[start]) << 24
		}
		out[base] = int32(kind | line<<8 | charBits)
		out[base+1] = int32(start)
		out[base+2] = int32(start + size)
	}
	if r.pos != r.end {
		return nil, false
	}
	return out, true
}

func renvoUnitUsesPanic(p *renvoProgram) bool {
	renvoNonNil(p)
	for i := 0; i < renvoTokCount(p); i++ {
		if renvoTokIdentIs(p, i, "defer") || renvoTokIdentIs(p, i, "panic") || renvoTokIdentIs(p, i, "recover") || renvoTokCharIs(p, i, '.') && renvoTokCharIs(p, i+1, '(') {
			return true
		}
	}
	return false
}

func renvoDecodeUnitProgram(src []byte) (renvoProgram, bool, bool) {
	var prog renvoProgram
	if len(src) < 4 {
		return prog, false, true
	}
	if src[0] != renvoUnitMagic[0] || src[1] != renvoUnitMagic[1] || src[2] != renvoUnitMagic[2] || src[3] != renvoUnitMagic[3] {
		return prog, false, true
	}
	if len(src) < 14 {
		return prog, true, false
	}
	if int(src[4])|(int(src[5])<<8) != renvoUnitVersion {
		return prog, true, false
	}
	if int(src[6])|(int(src[7])<<8) != 0 {
		return prog, true, false
	}
	length := renvoUnitRead32(src, 10)
	if int(src[8])|(int(src[9])<<8) != renvoUnitTagUnit || length < 0 {
		return prog, true, false
	}
	rootStart := 14
	rootEnd := rootStart + length
	if rootEnd != len(src) || rootEnd < rootStart {
		return prog, true, false
	}
	var text []byte
	var tokenData []byte
	var declData []byte
	var funcData []byte
	seen := 0
	pos := rootStart
	for pos < rootEnd {
		if pos+6 > rootEnd {
			return prog, true, false
		}
		tag := int(src[pos]) | (int(src[pos+1]) << 8)
		length := renvoUnitRead32(src, pos+2)
		pos = pos + 6
		if length < 0 {
			return prog, true, false
		}
		next := pos + length
		if next < pos || next > rootEnd {
			return prog, true, false
		}
		if tag == renvoUnitTagUnit {
			return prog, true, false
		}
		knownTag := tag == renvoUnitTagPackage || tag == renvoUnitTagImportPath || tag >= renvoUnitTagText && tag <= renvoUnitTagStmts
		if knownTag {
			bit := 1 << tag
			if seen&bit != 0 {
				return prog, true, false
			}
			seen = seen | bit
		}
		if tag == renvoUnitTagPackage {
			if length == 0 {
				return prog, true, false
			}
		}
		if tag == renvoUnitTagText {
			text = src[pos:next]
		}
		if tag == renvoUnitTagTokens {
			tokenData = src[pos:next]
		}
		if tag == renvoUnitTagDecls {
			declData = src[pos:next]
		}
		if tag == renvoUnitTagFuncs {
			funcData = src[pos:next]
		}
		pos = next
	}
	required := 1<<renvoUnitTagPackage | 1<<renvoUnitTagImportPath | 1<<renvoUnitTagText | 1<<renvoUnitTagTokens | 1<<renvoUnitTagDecls | 1<<renvoUnitTagFuncs
	if seen&required != required {
		return prog, true, false
	}
	if len(text) == 0 || len(tokenData) == 0 {
		return prog, true, false
	}
	tokens, tokensOK := renvoDecodeUnitTokens(text, tokenData)
	if !tokensOK {
		return prog, true, false
	}
	tokenCount := len(tokens) / renvoTokenStride
	if tokenCount <= 0 {
		return prog, true, false
	}
	if int(tokens[(tokenCount-1)*renvoTokenStride])&255 != renvoTokEOF {
		return prog, true, false
	}
	prog.src = text
	prog.toks.data = tokens
	prog.toks.count = tokenCount
	prog.toks.panicEnabled = renvoUnitUsesPanic(&prog)
	declReader := renvoUnitReader{src: declData, end: len(declData), ok: true}
	declCount := renvoUnitReadVar(&declReader)
	if !declReader.ok {
		return prog, true, false
	}
	prog.decls = make([]renvoDecl, 0, declCount)
	for i := 0; i < declCount; i++ {
		var decl renvoDecl
		nameSize := 0
		tokCount := 0
		decl.kind = renvoUnitReadVar(&declReader)
		decl.nameStart = renvoUnitReadVar(&declReader)
		nameSize = renvoUnitReadVar(&declReader)
		decl.startTok = renvoUnitReadVar(&declReader)
		tokCount = renvoUnitReadVar(&declReader)
		if !declReader.ok {
			return prog, true, false
		}
		decl.nameEnd = decl.nameStart + nameSize
		decl.endTok = decl.startTok + tokCount
		if !renvoUnitValidRange(len(text), decl.nameStart, decl.nameEnd) || !renvoUnitValidTokenRange(tokenCount, decl.startTok, decl.endTok) {
			return prog, true, false
		}
		prog.decls = append(prog.decls, decl)
	}
	if declReader.pos != declReader.end {
		return prog, true, false
	}
	funcReader := renvoUnitReader{src: funcData, end: len(funcData), ok: true}
	funcCount := renvoUnitReadVar(&funcReader)
	if !funcReader.ok {
		return prog, true, false
	}
	prog.funcs = make([]renvoFuncDecl, 0, funcCount)
	for i := 0; i < funcCount; i++ {
		var fn renvoFuncDecl
		nameSize := 0
		nameTokDelta := 0
		receiverCount := 0
		bodyCount := 0
		endCount := 0
		fn.nameStart = renvoUnitReadVar(&funcReader)
		nameSize = renvoUnitReadVar(&funcReader)
		fn.startTok = renvoUnitReadVar(&funcReader)
		nameTokDelta = renvoUnitReadVar(&funcReader)
		fn.receiverStart = renvoUnitReadVar(&funcReader)
		receiverCount = renvoUnitReadVar(&funcReader)
		fn.bodyStart = renvoUnitReadVar(&funcReader)
		bodyCount = renvoUnitReadVar(&funcReader)
		endCount = renvoUnitReadVar(&funcReader)
		if !funcReader.ok {
			return prog, true, false
		}
		fn.nameEnd = fn.nameStart + nameSize
		fn.nameTok = fn.startTok + nameTokDelta
		fn.receiverEnd = fn.receiverStart + receiverCount
		fn.bodyEnd = fn.bodyStart + bodyCount
		fn.endTok = fn.bodyEnd + endCount
		if !renvoUnitValidRange(len(text), fn.nameStart, fn.nameEnd) || !renvoUnitValidTokenRange(tokenCount, fn.startTok, fn.endTok) {
			return prog, true, false
		}
		if fn.nameTok < 0 || fn.nameTok >= tokenCount || fn.bodyStart < 0 || fn.bodyEnd >= tokenCount || fn.bodyStart > fn.bodyEnd {
			return prog, true, false
		}
		prog.funcs = append(prog.funcs, fn)
	}
	if funcReader.pos != funcReader.end {
		return prog, true, false
	}
	prog.ok = true
	return prog, true, true
}

func renvoUnitValidRange(limit int, start int, end int) bool {
	if start < 0 || end < start {
		return false
	}
	return end <= limit
}

func renvoUnitValidTokenRange(limit int, start int, end int) bool {
	if start < 0 || end < start {
		return false
	}
	return end <= limit
}

func renvoCompileProgramToOutput(prog *renvoProgram, output int, target int, arenaSize int) int {
	renvoNonNil(prog)
	renvoFixedTarget = target
	renvoSetTarget(target)
	if !prog.ok {
		renvoPrintErr("renvo: parse failed\n")
		return 1
	}
	var meta renvoMeta
	renvoBuildMetaInto(prog, &meta)
	if !meta.ok {
		renvoPrintErr("renvo: meta failed\n")
		return 1
	}
	meta.arenaSize = renvoResolveArenaSize(target, arenaSize)
	var result renvoCompileResult
	if renvoFixedTarget == renvoTargetLinux386 || renvoFixedTarget == renvoTargetWindows386 {
		result = renvoTryCompileScalarProgram386(prog, &meta)
	} else if renvoFixedTarget == renvoTargetLinuxAarch64 || renvoFixedTarget == renvoTargetDarwinArm64 || renvoFixedTarget == renvoTargetWindowsArm64 {
		result = renvoTryCompileScalarProgramAarch64(prog, &meta)
	} else if renvoFixedTarget == renvoTargetLinuxArm {
		result = renvoTryCompileScalarProgramArm(prog, &meta)
	} else if renvoFixedTarget == renvoTargetWasiWasm32 {
		result = renvoTryCompileScalarProgramWasm32(prog, &meta)
	} else if renvoFixedTarget != 0 {
		result = renvoTryCompileScalarProgramAmd64(prog, &meta)
	} else if target == renvoTargetLinux386 || target == renvoTargetWindows386 {
		result = renvoTryCompileScalarProgram386(prog, &meta)
	} else if target == renvoTargetLinuxAarch64 || target == renvoTargetDarwinArm64 || target == renvoTargetWindowsArm64 {
		result = renvoTryCompileScalarProgramAarch64(prog, &meta)
	} else if target == renvoTargetLinuxArm {
		result = renvoTryCompileScalarProgramArm(prog, &meta)
	} else if target == renvoTargetWasiWasm32 {
		result = renvoTryCompileScalarProgramWasm32(prog, &meta)
	} else {
		result = renvoTryCompileScalarProgramAmd64(prog, &meta)
	}
	if result.ok {
		write(output, result.data, -1)
		return 0
	}
	renvoPrintErr("renvo: compilation failed\n")
	return 1
}

func renvoCompileUnitInput(input []int, output int, target int, arenaSize int) int {
	if len(input) != 1 {
		return -1
	}
	if input[0] == 0 {
		var src []byte
		src = renvoReadAll(input[0], src)
		if len(src) >= 4 && src[0] == 'R' && src[1] == 'N' && src[2] == 'V' && src[3] == 'O' {
			prog, isUnit, ok := renvoDecodeUnitProgram(src)
			if !isUnit {
				return -1
			}
			if !ok {
				renvoPrintErr("renvo: invalid unit input\n")
				return 1
			}
			return renvoCompileProgramToOutput(&prog, output, target, arenaSize)
		}
		prog := renvoParseProgram(src)
		return renvoCompileProgramToOutput(&prog, output, target, arenaSize)
	}
	header := make([]byte, 4)
	n := read(input[0], header, 0)
	if n != 4 || header[0] != 'R' || header[1] != 'N' || header[2] != 'V' || header[3] != 'O' {
		return -1
	}
	var unit []byte
	unit = renvoReadAll(input[0], unit)
	prog, isUnit, ok := renvoDecodeUnitProgram(unit)
	if !isUnit {
		return -1
	}
	if !ok {
		renvoPrintErr("renvo: invalid unit input\n")
		return 1
	}
	return renvoCompileProgramToOutput(&prog, output, target, arenaSize)
}

func appMain(args []string, env []string) int {
	input := make([]int, 256)
	inputCount := 0
	var outputPath string
	target := renvoDefaultTarget
	arenaSize := 0
	renvoCompilerStripSymbols = false
	renvoCompilerWindowsSubsystem = 3
	if len(args) == 0 {
		renvoPrintErr("renvo: missing output path (-o)\n")
		renvoPrintUsage()
		return 1
	}
	i := 1
	for i != len(args) {
		arg := args[i]
		if len(arg) == 2 && arg[0] == '-' && arg[1] == 's' {
			renvoCompilerStripSymbols = true
			i++
			continue
		}
		if len(arg) == 12 && arg[0] == '-' && arg[1] == 'w' && arg[2] == 'i' && arg[3] == 'n' && arg[4] == 'd' && arg[5] == 'o' && arg[6] == 'w' && arg[7] == 's' && arg[8] == '-' && arg[9] == 'g' && arg[10] == 'u' && arg[11] == 'i' {
			renvoCompilerWindowsSubsystem = 2
			i++
			continue
		}
		if len(arg) == 2 && arg[0] == '-' && arg[1] == 'o' {
			i++
			if i == len(args) {
				renvoPrintErr("renvo: missing argument for -o\n")
				renvoPrintUsage()
				return 1
			}
			outputArg := args[i]
			outputPath = outputArg
			i++
			continue
		}
		if len(arg) == 2 && arg[0] == '-' && arg[1] == 't' {
			i++
			if i == len(args) {
				renvoPrintErr("renvo: missing argument for -t\n")
				renvoPrintUsage()
				return 1
			}
			targetArg := args[i]
			target = renvoParseTargetArg(targetArg)
			if target == 0 {
				renvoPrintUnsupportedTarget(targetArg)
				return 1
			}
			i++
			continue
		}
		if len(arg) == 11 && arg[0] == '-' && arg[1] == 'a' && arg[2] == 'r' && arg[3] == 'e' && arg[4] == 'n' && arg[5] == 'a' && arg[6] == '-' && arg[7] == 's' && arg[8] == 'i' && arg[9] == 'z' && arg[10] == 'e' {
			i++
			if i == len(args) {
				renvoPrintErr("renvo: missing argument for -arena-size\n")
				renvoPrintUsage()
				return 1
			}
			parsedArenaSize, ok := renvoParsePositiveDecimal(args[i])
			if !ok {
				renvoPrintErr("renvo: invalid arena size: ")
				renvoPrintErr(args[i])
				renvoPrintErr("\n")
				return 1
			}
			arenaSize = parsedArenaSize
			i++
			continue
		}
		if len(arg) == 1 && arg[0] == '-' {
			if inputCount == len(input) {
				renvoPrintErr("renvo: too many input files\n")
				return 1
			}
			input[inputCount] = 0
			inputCount++
			i++
			continue
		}
		if len(arg) > 0 {
			if arg[0] == '-' {
				renvoPrintErr("renvo: unknown option: ")
				renvoPrintErr(arg)
				renvoPrintErr("\n")
				renvoPrintUsage()
				return 1
			}
		}
		fd := renvoOpenArg(arg, env)
		if fd < 0 {
			renvoPrintErr("renvo: failed to open input: ")
			renvoPrintErr(arg)
			renvoPrintErr("\n")
			return 1
		}
		if inputCount == len(input) {
			renvoPrintErr("renvo: too many input files\n")
			return 1
		}
		input[inputCount] = fd
		inputCount++
		i++
	}
	if outputPath == "" {
		renvoPrintErr("renvo: missing output path (-o)\n")
		renvoPrintUsage()
		return 1
	}
	if inputCount == 0 {
		renvoPrintErr("renvo: no input files\n")
		renvoPrintUsage()
		return 1
	}
	if renvoCompilerWindowsSubsystem == 2 && target != renvoTargetWindowsAmd64 && target != renvoTargetWindows386 && target != renvoTargetWindowsArm64 {
		renvoPrintErr("renvo: -windows-gui requires a Windows target\n")
		return 1
	}
	output := 1
	if outputPath != "-" {
		output = open(outputPath, O_RDWR|O_CREATE|O_TRUNC)
		if output < 0 {
			renvoPrintErr("renvo: failed to open output: ")
			renvoPrintErr(outputPath)
			renvoPrintErr("\n")
			return 1
		}
	}
	unitResult := renvoCompileUnitInput(input[:inputCount], output, target, arenaSize)
	if unitResult >= 0 {
		return unitResult
	}
	return compileTarget(input[:inputCount], output, target, arenaSize)
}

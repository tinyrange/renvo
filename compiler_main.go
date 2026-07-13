package main

var rtgCompilerDefaultTarget int = rtgTargetLinuxAmd64
var rtgCompilerFixedTarget int
var rtgCompilerStripSymbols bool

func rtgOpenArg(path string, env []string) int {
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

func rtgParseTargetArg(target string) int {
	if len(target) == 11 && target[0] == 'l' && target[1] == 'i' && target[2] == 'n' && target[3] == 'u' && target[4] == 'x' && target[5] == '/' && target[6] == 'a' && target[7] == 'm' && target[8] == 'd' && target[9] == '6' && target[10] == '4' {
		return rtgTargetLinuxAmd64
	}
	if len(target) == 9 && target[0] == 'l' && target[1] == 'i' && target[2] == 'n' && target[3] == 'u' && target[4] == 'x' && target[5] == '/' && target[6] == '3' && target[7] == '8' && target[8] == '6' {
		return rtgTargetLinux386
	}
	if len(target) == 13 && target[0] == 'l' && target[1] == 'i' && target[2] == 'n' && target[3] == 'u' && target[4] == 'x' && target[5] == '/' && target[6] == 'a' && target[7] == 'a' && target[8] == 'r' && target[9] == 'c' && target[10] == 'h' && target[11] == '6' && target[12] == '4' {
		return rtgTargetLinuxAarch64
	}
	if len(target) == 9 && target[0] == 'l' && target[1] == 'i' && target[2] == 'n' && target[3] == 'u' && target[4] == 'x' && target[5] == '/' && target[6] == 'a' && target[7] == 'r' && target[8] == 'm' {
		return rtgTargetLinuxArm
	}
	if len(target) == 13 && target[0] == 'w' && target[1] == 'i' && target[2] == 'n' && target[3] == 'd' && target[4] == 'o' && target[5] == 'w' && target[6] == 's' && target[7] == '/' && target[8] == 'a' && target[9] == 'm' && target[10] == 'd' && target[11] == '6' && target[12] == '4' {
		return rtgTargetWindowsAmd64
	}
	if len(target) == 11 && target[0] == 'w' && target[1] == 'i' && target[2] == 'n' && target[3] == 'd' && target[4] == 'o' && target[5] == 'w' && target[6] == 's' && target[7] == '/' && target[8] == '3' && target[9] == '8' && target[10] == '6' {
		return rtgTargetWindows386
	}
	if len(target) == 11 && target[0] == 'w' && target[1] == 'a' && target[2] == 's' && target[3] == 'i' && target[4] == '/' && target[5] == 'w' && target[6] == 'a' && target[7] == 's' && target[8] == 'm' && target[9] == '3' && target[10] == '2' {
		return rtgTargetWasiWasm32
	}
	if len(target) == 12 && target[0] == 'd' && target[1] == 'a' && target[2] == 'r' && target[3] == 'w' && target[4] == 'i' && target[5] == 'n' && target[6] == '/' && target[7] == 'a' && target[8] == 'r' && target[9] == 'm' && target[10] == '6' && target[11] == '4' {
		return rtgTargetDarwinArm64
	}
	return 0
}

func rtgPrintErr(s string) {
	write(2, []byte(s), -1)
}

func rtgPrintIntErr(v int) {
	if v == 0 {
		rtgPrintErr("0")
		return
	}
	if v < 0 {
		rtgPrintErr("-")
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

func rtgPrintUsage() {
	rtgPrintErr("usage: rtg [-s] [-t linux/amd64|linux/386|linux/aarch64|linux/arm|windows/amd64|windows/386|wasi/wasm32|darwin/arm64] -o <output|-> <input.go|->...\n")
}

func rtgPrintUnsupportedTarget(target string) {
	rtgPrintErr("rtg: unsupported target: ")
	rtgPrintErr(target)
	rtgPrintErr("\n")
	rtgPrintErr("rtg: supported targets: linux/amd64, linux/386, linux/aarch64, linux/arm, windows/amd64, windows/386, wasi/wasm32, darwin/arm64\n")
}

func rtgUnitRead32(src []byte, pos int) int {
	return int(src[pos]) | (int(src[pos+1]) << 8) | (int(src[pos+2]) << 16) | (int(src[pos+3]) << 24)
}

func rtgUnitReadVar(src []byte, pos int, end int) (int, int, bool) {
	value := 0
	shift := 0
	for pos < end && shift <= 28 {
		b := src[pos]
		pos++
		value = value | (int(b&0x7f) << shift)
		if b < 0x80 {
			return value, pos, true
		}
		shift = shift + 7
	}
	return 0, pos, false
}

func rtgDecodeUnitTokens(text []byte, data []byte) ([]byte, bool) {
	pos := 0
	count, next, ok := rtgUnitReadVar(data, pos, len(data))
	if !ok {
		return nil, false
	}
	pos = next
	out := make([]byte, 0, count*rtgTokenStride)
	start := 0
	line := 0
	for i := 0; i < count; i++ {
		kind, next, ok := rtgUnitReadVar(data, pos, len(data))
		if !ok {
			return nil, false
		}
		pos = next
		delta, next, ok := rtgUnitReadVar(data, pos, len(data))
		if !ok {
			return nil, false
		}
		pos = next
		size, next, ok := rtgUnitReadVar(data, pos, len(data))
		if !ok {
			return nil, false
		}
		pos = next
		lineDelta, next, ok := rtgUnitReadVar(data, pos, len(data))
		if !ok {
			return nil, false
		}
		pos = next
		start = start + delta
		line = line + lineDelta
		if kind < 0 || kind > 255 || start < 0 || start > 0xffffff || size < 0 || line < 0 || line > 0xffff || start+size > len(text) {
			return nil, false
		}
		if kind == rtgTokOp {
			if size > 255 {
				return nil, false
			}
		} else if size > 0xffff {
			return nil, false
		}
		base := len(out)
		out = out[:base+rtgTokenStride]
		out[base] = byte(kind)
		out[base+1] = byte(start)
		out[base+2] = byte(start >> 8)
		out[base+3] = byte(start >> 16)
		out[base+4] = byte(size)
		if kind == rtgTokOp {
			if size > 0 {
				out[base+5] = text[start]
			} else {
				out[base+5] = 0
			}
		} else {
			out[base+5] = byte(size >> 8)
		}
		out[base+6] = byte(line)
		out[base+7] = byte(line >> 8)
	}
	if pos != len(data) {
		return nil, false
	}
	return out, true
}

func rtgDecodeUnitProgram(src []byte) (rtgProgram, bool, bool) {
	var prog rtgProgram
	if len(src) < 4 {
		return prog, false, true
	}
	if src[0] != 'R' || src[1] != 'T' || src[2] != 'G' || src[3] != 'U' {
		return prog, false, true
	}
	if len(src) < 14 {
		return prog, true, false
	}
	if int(src[4])|(int(src[5])<<8) != 1 {
		return prog, true, false
	}
	length := rtgUnitRead32(src, 10)
	if int(src[8])|(int(src[9])<<8) != 1 || length < 0 {
		return prog, true, false
	}
	rootStart := 14
	rootEnd := rootStart + length
	if rootEnd > len(src) || rootEnd < rootStart {
		return prog, true, false
	}
	var text []byte
	var tokenData []byte
	var declData []byte
	var funcData []byte
	pos := rootStart
	for pos < rootEnd {
		if pos+6 > rootEnd {
			return prog, true, false
		}
		tag := int(src[pos]) | (int(src[pos+1]) << 8)
		length := rtgUnitRead32(src, pos+2)
		pos = pos + 6
		if length < 0 {
			return prog, true, false
		}
		next := pos + length
		if next < pos || next > rootEnd {
			return prog, true, false
		}
		if tag == 7 {
			text = src[pos:next]
		}
		if tag == 8 {
			tokenData = src[pos:next]
		}
		if tag == 9 {
			declData = src[pos:next]
		}
		if tag == 10 {
			funcData = src[pos:next]
		}
		pos = next
	}
	if len(text) == 0 || len(tokenData) == 0 {
		return prog, true, false
	}
	tokens, tokensOK := rtgDecodeUnitTokens(text, tokenData)
	if !tokensOK {
		return prog, true, false
	}
	tokenCount := len(tokens) / rtgTokenStride
	if tokenCount <= 0 {
		return prog, true, false
	}
	if int(tokens[(tokenCount-1)*rtgTokenStride]) != rtgTokEOF {
		return prog, true, false
	}
	prog.src = text
	prog.toks.data = tokens
	declCount, next, ok := rtgUnitReadVar(declData, 0, len(declData))
	if !ok {
		return prog, true, false
	}
	pos = next
	prog.decls = make([]rtgDecl, 0, declCount)
	for i := 0; i < declCount; i++ {
		var decl rtgDecl
		nameSize := 0
		tokCount := 0
		decl.kind, pos, ok = rtgUnitReadVar(declData, pos, len(declData))
		if !ok {
			return prog, true, false
		}
		decl.nameStart, pos, ok = rtgUnitReadVar(declData, pos, len(declData))
		if !ok {
			return prog, true, false
		}
		nameSize, pos, ok = rtgUnitReadVar(declData, pos, len(declData))
		if !ok {
			return prog, true, false
		}
		decl.startTok, pos, ok = rtgUnitReadVar(declData, pos, len(declData))
		if !ok {
			return prog, true, false
		}
		tokCount, pos, ok = rtgUnitReadVar(declData, pos, len(declData))
		if !ok {
			return prog, true, false
		}
		decl.nameEnd = decl.nameStart + nameSize
		decl.endTok = decl.startTok + tokCount
		if !rtgUnitValidRange(len(text), decl.nameStart, decl.nameEnd) || !rtgUnitValidTokenRange(tokenCount, decl.startTok, decl.endTok) {
			return prog, true, false
		}
		prog.decls = append(prog.decls, decl)
	}
	if pos != len(declData) {
		return prog, true, false
	}
	funcCount, next, ok := rtgUnitReadVar(funcData, 0, len(funcData))
	if !ok {
		return prog, true, false
	}
	pos = next
	prog.funcs = make([]rtgFuncDecl, 0, funcCount)
	for i := 0; i < funcCount; i++ {
		var fn rtgFuncDecl
		nameSize := 0
		nameTokDelta := 0
		receiverCount := 0
		bodyCount := 0
		endCount := 0
		fn.nameStart, pos, ok = rtgUnitReadVar(funcData, pos, len(funcData))
		if !ok {
			return prog, true, false
		}
		nameSize, pos, ok = rtgUnitReadVar(funcData, pos, len(funcData))
		if !ok {
			return prog, true, false
		}
		fn.startTok, pos, ok = rtgUnitReadVar(funcData, pos, len(funcData))
		if !ok {
			return prog, true, false
		}
		nameTokDelta, pos, ok = rtgUnitReadVar(funcData, pos, len(funcData))
		if !ok {
			return prog, true, false
		}
		fn.receiverStart, pos, ok = rtgUnitReadVar(funcData, pos, len(funcData))
		if !ok {
			return prog, true, false
		}
		receiverCount, pos, ok = rtgUnitReadVar(funcData, pos, len(funcData))
		if !ok {
			return prog, true, false
		}
		fn.bodyStart, pos, ok = rtgUnitReadVar(funcData, pos, len(funcData))
		if !ok {
			return prog, true, false
		}
		bodyCount, pos, ok = rtgUnitReadVar(funcData, pos, len(funcData))
		if !ok {
			return prog, true, false
		}
		endCount, pos, ok = rtgUnitReadVar(funcData, pos, len(funcData))
		if !ok {
			return prog, true, false
		}
		fn.nameEnd = fn.nameStart + nameSize
		fn.nameTok = fn.startTok + nameTokDelta
		fn.receiverEnd = fn.receiverStart + receiverCount
		fn.bodyEnd = fn.bodyStart + bodyCount
		fn.endTok = fn.bodyEnd + endCount
		if !rtgUnitValidRange(len(text), fn.nameStart, fn.nameEnd) || !rtgUnitValidTokenRange(tokenCount, fn.startTok, fn.endTok) {
			return prog, true, false
		}
		if fn.nameTok < 0 || fn.nameTok >= tokenCount || fn.bodyStart < 0 || fn.bodyEnd >= tokenCount || fn.bodyStart > fn.bodyEnd {
			return prog, true, false
		}
		prog.funcs = append(prog.funcs, fn)
	}
	if pos != len(funcData) {
		return prog, true, false
	}
	prog.ok = true
	return prog, true, true
}

func rtgUnitValidRange(limit int, start int, end int) bool {
	if start < 0 || end < start {
		return false
	}
	return end <= limit
}

func rtgUnitValidTokenRange(limit int, start int, end int) bool {
	if start < 0 || end < start {
		return false
	}
	return end <= limit
}

func rtgCompileProgramToOutput(prog *rtgProgram, output int, target int) int {
	rtgCompilerFixedTarget = target
	rtgSetTarget(target)
	if !prog.ok {
		rtgPrintErr("rtg: parse failed\n")
		return 1
	}
	var meta rtgMeta
	rtgBuildMetaInto(prog, &meta)
	if !meta.ok {
		rtgPrintErr("rtg: meta failed\n")
		return 1
	}
	var result rtgCompileResult
	if target == rtgTargetLinux386 || target == rtgTargetWindows386 {
		result = rtgTryCompileScalarProgram386(prog, &meta)
	} else if target == rtgTargetLinuxAarch64 || target == rtgTargetDarwinArm64 {
		result = rtgTryCompileScalarProgramAarch64(prog, &meta)
	} else if target == rtgTargetLinuxArm {
		result = rtgTryCompileScalarProgramArm(prog, &meta)
	} else if target == rtgTargetWasiWasm32 {
		result = rtgTryCompileScalarProgramWasm32(prog, &meta)
	} else {
		result = rtgTryCompileScalarProgramAmd64(prog, &meta)
	}
	if result.ok {
		write(output, result.data, -1)
		return 0
	}
	rtgPrintErr("rtg: compilation failed\n")
	return 1
}

func rtgCompileUnitInput(input []int, output int, target int) int {
	if len(input) != 1 {
		return -1
	}
	if input[0] == 0 {
		var src []byte
		src = rtgReadAll(input[0], src)
		if len(src) >= 4 && src[0] == 'R' && src[1] == 'T' && src[2] == 'G' && src[3] == 'U' {
			prog, isUnit, ok := rtgDecodeUnitProgram(src)
			if !isUnit {
				return -1
			}
			if !ok {
				rtgPrintErr("rtg: invalid unit input\n")
				return 1
			}
			return rtgCompileProgramToOutput(&prog, output, target)
		}
		prog := rtgParseProgram(src)
		return rtgCompileProgramToOutput(&prog, output, target)
	}
	header := make([]byte, 4)
	n := read(input[0], header, 0)
	if n != 4 || header[0] != 'R' || header[1] != 'T' || header[2] != 'G' || header[3] != 'U' {
		return -1
	}
	var unit []byte
	unit = rtgReadAll(input[0], unit)
	prog, isUnit, ok := rtgDecodeUnitProgram(unit)
	if !isUnit {
		return -1
	}
	if !ok {
		rtgPrintErr("rtg: invalid unit input\n")
		return 1
	}
	return rtgCompileProgramToOutput(&prog, output, target)
}

func appMain(args []string, env []string) int {
	input := make([]int, 256)
	inputCount := 0
	var outputPath string
	target := rtgCompilerDefaultTarget
	if len(args) == 0 {
		rtgPrintErr("rtg: missing output path (-o)\n")
		rtgPrintUsage()
		return 1
	}
	i := 1
	for i != len(args) {
		arg := args[i]
		if len(arg) == 2 && arg[0] == '-' && arg[1] == 's' {
			rtgCompilerStripSymbols = true
			i++
			continue
		}
		if len(arg) == 2 && arg[0] == '-' && arg[1] == 'o' {
			i++
			if i == len(args) {
				rtgPrintErr("rtg: missing argument for -o\n")
				rtgPrintUsage()
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
				rtgPrintErr("rtg: missing argument for -t\n")
				rtgPrintUsage()
				return 1
			}
			targetArg := args[i]
			target = rtgParseTargetArg(targetArg)
			if target == 0 {
				rtgPrintUnsupportedTarget(targetArg)
				return 1
			}
			i++
			continue
		}
		if len(arg) == 1 && arg[0] == '-' {
			if inputCount == len(input) {
				rtgPrintErr("rtg: too many input files\n")
				return 1
			}
			input[inputCount] = 0
			inputCount++
			i++
			continue
		}
		if len(arg) > 0 {
			if arg[0] == '-' {
				rtgPrintErr("rtg: unknown option: ")
				rtgPrintErr(arg)
				rtgPrintErr("\n")
				rtgPrintUsage()
				return 1
			}
		}
		fd := rtgOpenArg(arg, env)
		if fd < 0 {
			rtgPrintErr("rtg: failed to open input: ")
			rtgPrintErr(arg)
			rtgPrintErr("\n")
			return 1
		}
		if inputCount == len(input) {
			rtgPrintErr("rtg: too many input files\n")
			return 1
		}
		input[inputCount] = fd
		inputCount++
		i++
	}
	if outputPath == "" {
		rtgPrintErr("rtg: missing output path (-o)\n")
		rtgPrintUsage()
		return 1
	}
	if inputCount == 0 {
		rtgPrintErr("rtg: no input files\n")
		rtgPrintUsage()
		return 1
	}
	output := 1
	if outputPath != "-" {
		output = open(outputPath, O_RDWR|O_CREATE|O_TRUNC)
		if output < 0 {
			rtgPrintErr("rtg: failed to open output: ")
			rtgPrintErr(outputPath)
			rtgPrintErr("\n")
			return 1
		}
	}
	unitResult := rtgCompileUnitInput(input[:inputCount], output, target)
	if unitResult >= 0 {
		return unitResult
	}
	return compileTarget(input[:inputCount], output, target)
}

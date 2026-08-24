package main

var renvoDefaultTarget int = renvoTargetLinuxAmd64
var renvoFixedTarget int = 0
var renvoCompilerStripSymbols bool
var renvoKernelRelease string
var renvoKernelModuleName string
var renvoKernelBTF []byte
var renvoKernelSymvers []byte
var renvoKernelVersion string
var renvoKernelModuleSize int
var renvoKernelModuleNameOff int
var renvoKernelModuleInitOff int
var renvoKernelModuleExitOff int
var renvoKernelLicense string

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
	renvoPrintErr("usage: renvo [options] [-emit-image] -o <output|-> <input.go|->...\n")
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
	renvoPrintErr("renvo: supported targets: ")
	renvoPrintErr(renvoSupportedTargets)
	renvoPrintErr("\n")
}

func appMain(args []string, env []string) int {
	input := make([]int, 256)
	inputCount := 0
	var outputPath string
	var moduleNamePath string
	moduleLicense := "Proprietary"
	target := renvoDefaultTarget
	arenaSize := 0
	renvoCompilerStripSymbols = false
	renvoCompilerEmitImage = false
	renvoCompilerObjectFile = false
	renvoCompilerCode16 = false
	renvoCompilerRegParm = 0
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
		if arg == "-emit-image" {
			renvoCompilerEmitImage = true
			i++
			continue
		}
		if arg == "-object" {
			renvoCompilerObjectFile = true
			i++
			continue
		}
		if arg == "-code16" {
			renvoCompilerCode16 = true
			i++
			continue
		}
		if arg == "-regparm3" {
			renvoCompilerRegParm = 3
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
		if arg == "-module-name" {
			i++
			if i == len(args) {
				renvoPrintErr("renvo: missing argument for -module-name\n")
				return 1
			}
			moduleNamePath = args[i]
			i++
			continue
		}
		if arg == "-module-license" {
			i++
			if i == len(args) || args[i] == "" {
				renvoPrintErr("renvo: missing argument for -module-license\n")
				return 1
			}
			moduleLicense = args[i]
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
	renvoKernelModuleName = renvoKernelNameFromOutput(outputPath)
	if moduleNamePath != "" {
		renvoKernelModuleName = renvoKernelNameFromOutput(moduleNamePath)
	}
	renvoKernelLicense = moduleLicense
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

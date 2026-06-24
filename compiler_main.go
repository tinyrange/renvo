package main

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
	if target == "linux/amd64" {
		return rtgTargetLinuxAmd64
	}
	if target == "linux/386" {
		return rtgTargetLinux386
	}
	if target == "linux/aarch64" {
		return rtgTargetLinuxAarch64
	}
	if target == "linux/arm" {
		return rtgTargetLinuxArm
	}
	if target == "windows/amd64" {
		return rtgTargetWindowsAmd64
	}
	if target == "windows/386" {
		return rtgTargetWindows386
	}
	if target == "wasi/wasm32" {
		return rtgTargetWasiWasm32
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
	rtgPrintErr("usage: rtg [-t linux/amd64|linux/386|linux/aarch64|linux/arm|windows/amd64|windows/386|wasi/wasm32] -o <output> <input.go>...\n")
}

func rtgPrintUnsupportedTarget(target string) {
	rtgPrintErr("rtg: unsupported target: ")
	rtgPrintErr(target)
	rtgPrintErr("\n")
	rtgPrintErr("rtg: supported targets: linux/amd64, linux/386, linux/aarch64, linux/arm, windows/amd64, windows/386, wasi/wasm32\n")
}

func appMain(args []string, env []string) int {
	var input []int
	var outputPath string
	target := rtgTargetLinuxAmd64
	i := 1
	for i < len(args) {
		arg := args[i]
		if arg == "-o" {
			i++
			if i >= len(args) {
				rtgPrintErr("rtg: missing argument for -o\n")
				rtgPrintUsage()
				return 1
			}
			outputArg := args[i]
			outputPath = outputArg
			i++
			continue
		}
		if arg == "-t" {
			i++
			if i >= len(args) {
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
		input = append(input, fd)
		i++
	}
	if outputPath == "" {
		rtgPrintErr("rtg: missing output path (-o)\n")
		rtgPrintUsage()
		return 1
	}
	if len(input) == 0 {
		rtgPrintErr("rtg: no input files\n")
		rtgPrintUsage()
		return 1
	}
	var output int = open(outputPath, O_RDWR|O_CREATE|O_TRUNC)
	if output < 0 {
		rtgPrintErr("rtg: failed to open output: ")
		rtgPrintErr(outputPath)
		rtgPrintErr("\n")
		return 1
	}
	return compileTarget(input, output, target)
}

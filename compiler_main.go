package main

func rtgOpenArg(path string, env []string) int {
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
	return 0
}

func rtgPrintErr(s string) {
	write(2, []byte(s), -1)
}

func rtgPrintUsage() {
	rtgPrintErr("usage: rtg [-t linux/amd64|linux/386] -o <output> <input.go>...\n")
}

func rtgPrintUnsupportedTarget(target string) {
	rtgPrintErr("rtg: unsupported target: ")
	rtgPrintErr(target)
	rtgPrintErr("\n")
	rtgPrintErr("rtg: supported targets: linux/amd64, linux/386\n")
}

func appMain(args []string, env []string) int {
	var input []int
	var outputPath string
	target := rtgTargetLinuxAmd64
	i := 1
	for i < len(args) {
		if args[i] == "-o" {
			i++
			if i >= len(args) {
				rtgPrintErr("rtg: missing argument for -o\n")
				rtgPrintUsage()
				return 1
			}
			outputPath = args[i]
			i++
			continue
		}
		if args[i] == "-t" {
			i++
			if i >= len(args) {
				rtgPrintErr("rtg: missing argument for -t\n")
				rtgPrintUsage()
				return 1
			}
			target = rtgParseTargetArg(args[i])
			if target == 0 {
				rtgPrintUnsupportedTarget(args[i])
				return 1
			}
			i++
			continue
		}
		if len(args[i]) > 0 {
			if args[i][0] == '-' {
				rtgPrintErr("rtg: unknown option: ")
				rtgPrintErr(args[i])
				rtgPrintErr("\n")
				rtgPrintUsage()
				return 1
			}
		}
		fd := rtgOpenArg(args[i], env)
		if fd < 0 {
			rtgPrintErr("rtg: failed to open input: ")
			rtgPrintErr(args[i])
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
	return compileLinuxTarget(input, output, target)
}

//go:build rtg

package driver

import (
	"j5.nz/rtg/rtg/internal/arena"
	"j5.nz/rtg/rtg/internal/backendbridge"
	"j5.nz/rtg/rtg/internal/load"
)

const rtgGetdents64LinuxAmd64 = 217
const rtgGetdents64LinuxAarch64 = 61
const rtgGetdents64Linux386 = 220

type RTGFS struct{}

func syscall(num int, fd int, buf []byte, size int) int { return 0 }

func RunRTGCommand(args []string, env []string) int {
	if CommandHelpRequested(args) {
		print(HelpText)
		return 0
	}
	commandArgs := args
	if len(commandArgs) > 0 {
		commandArgs = commandArgs[1:]
	}
	resetArena := rtgFrontendCanResetArena()
	mark := 0
	if resetArena {
		mark = arena.Mark()
	}
	built := buildFromFSCompact(commandArgs, rtgWorkDir(env), rtgStdRoot(args, env), RTGFS{})
	if !built.Ok {
		if resetArena {
			arena.Reset(mark)
		}
		printRTGBuildError(built)
		return 1
	}
	unit := built.Unit
	target := built.Options.Target
	output := built.Options.Output
	strip := built.Options.Strip
	persistMark := 0
	if resetArena {
		persistMark = arena.PersistMark()
		unit = arena.PersistBytes(unit)
		target = arena.PersistString(target)
		output = arena.PersistString(output)
		backendMark := mark
		remainder := backendMark % 4096
		if remainder != 0 {
			backendMark = backendMark + 4096 - remainder
		}
		arena.Reset(backendMark)
	}
	ok := backendbridge.CompileUnitToOutputStripEnv(unit, target, output, strip, args, env)
	if resetArena {
		arena.PersistReset(persistMark)
	}
	if !ok {
		print("rtg: backend compilation failed\n")
		return 1
	}
	return 0
}

func rtgWorkDir(env []string) string {
	value := rtgEnvValue(env, "PWD")
	if value != "" {
		return value
	}
	return "."
}

func rtgStdRoot(args []string, env []string) string {
	value := rtgEnvValue(env, "RTG_STDROOT")
	if value != "" {
		return value
	}
	if rtgBundledStdEnabled {
		return "/std"
	}
	bundled := rtgBundledStdRoot(args)
	if bundled != "" {
		return bundled
	}
	return "/std"
}

func rtgEnvValue(env []string, key string) string {
	for i := 0; i < len(env); i++ {
		item := env[i]
		if len(item) <= len(key) || item[len(key)] != '=' {
			continue
		}
		matched := true
		for j := 0; j < len(key); j++ {
			if item[j] != key[j] {
				matched = false
				break
			}
		}
		if matched {
			return item[len(key)+1:]
		}
	}
	return ""
}

func rtgBundledStdRoot(args []string) string {
	dir := rtgExecutableDir(args)
	if dir == "" {
		return ""
	}
	path := load.JoinPath(dir, "std")
	if rtgPathExists(path) {
		return path
	}
	path = load.JoinPath(load.JoinPath(dir, ".."), "std")
	if rtgPathExists(path) {
		return path
	}
	path = load.JoinPath(load.JoinPath(load.JoinPath(dir, ".."), "share"), "rtg")
	path = load.JoinPath(path, "std")
	if rtgPathExists(path) {
		return path
	}
	return ""
}

func rtgExecutableDir(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return load.DirPath(args[0])
}

func rtgPathExists(path string) bool {
	fd := open(rtgPathCString(path), 0)
	if fd < 0 {
		return false
	}
	close(fd)
	return true
}

func (fs RTGFS) ReadFile(path string) ([]byte, bool) {
	if data, bundled := bundledStdReadFile(path); bundled {
		return data, true
	}
	fd := open(rtgPathCString(path), 0)
	if fd < 0 {
		return nil, false
	}
	buf := make([]byte, 4096)
	out := make([]byte, 0, len(buf))
	for {
		n := read(fd, buf, -1)
		if n < 0 {
			close(fd)
			return nil, false
		}
		if n == 0 {
			break
		}
		for i := 0; i < n; i++ {
			out = append(out, buf[i])
		}
	}
	close(fd)
	return out, true
}

func (fs RTGFS) ReadDir(path string) ([]DirEntry, bool) {
	if entries, bundled := bundledStdReadDir(path); bundled {
		return entries, true
	}
	return rtgReadDirNative(path)
}

func rtgDirNameIsDot(buf []byte, start int, end int) bool {
	size := end - start
	return size == 1 && buf[start] == '.' || size == 2 && buf[start] == '.' && buf[start+1] == '.'
}

func printRTGBuildError(result BuildResult) {
	if result.Error == BuildErrOptions {
		printOptionError(result.Options)
		return
	}
	if result.Error == BuildErrSource {
		messages := []string{
			"source error at ",
			"missing module at ",
			"invalid module at ",
			"bad package: ",
			"directory read failed: ",
			"file read failed: ",
			"bad build constraint: ",
			"source parse failed: ",
			"unresolved import: ",
		}
		err := result.Sources.Error
		if err < 1 || err >= len(messages) {
			err = 0
		}
		print("rtg: ")
		print(messages[err])
		print(result.ErrorPath)
		print("\n")
		return
	}
	if result.Error == BuildErrPipeline {
		print("rtg: frontend pipeline failed at package=")
		rtgPrintInt(result.ErrorPackage)
		print(" file=")
		rtgPrintInt(result.ErrorFile)
		print(" token=")
		rtgPrintInt(result.ErrorToken)
		print("\n")
		return
	}
	print("rtg: build failed\n")
}

func rtgPrintInt(value int) {
	if value == 0 {
		print("0")
		return
	}
	if value < 0 {
		print("-")
		value = -value
	}
	var digits []byte
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value = value / 10
	}
	for i := len(digits) - 1; i >= 0; i-- {
		print(string(digits[i : i+1]))
	}
}

func printOptionError(options Options) {
	if options.Error == ParseErrMissingOutput {
		print("rtg: missing output after -o\n")
		return
	}
	if options.Error == ParseErrMissingTarget {
		print("rtg: missing target after -t\n")
		return
	}
	if options.Error == ParseErrUnsupportedTarget {
		print("rtg: unsupported target: ")
		print(options.ErrorArg)
		print("\n")
		return
	}
	if options.Error == ParseErrUnknownOption {
		print("rtg: unknown option: ")
		print(options.ErrorArg)
		print("\n")
		return
	}
	if options.Error == ParseErrMissingTags {
		print("rtg: missing tags after -tags\n")
		return
	}
	if options.Error == ParseErrInvalidTags {
		print("rtg: invalid build tags: ")
		print(options.ErrorArg)
		print("\n")
		return
	}
	if options.Error == ParseErrMissingPackage {
		print("rtg: missing package path\n")
		return
	}
	if options.Error == ParseErrExtraPackage {
		print("rtg: extra package path: ")
		print(options.ErrorArg)
		print("\n")
		return
	}
	print("rtg: option parse failed\n")
}

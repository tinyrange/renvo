//go:build rtg

package driver

import "j5.nz/rtg/rtg/internal/backendbridge"

const rtgGetdents64LinuxAmd64 = 217
const rtgGetdents64LinuxAarch64 = 61
const rtgGetdents64Linux386 = 220

type RTGFS struct{}

func syscall(num int, fd int, buf []byte, size int) int { return 0 }

func rtg_runtime_ArenaMark() int { return 0 }

func rtg_runtime_ArenaReset(mark int) {}

func rtg_runtime_ArenaPersistString(value string) string { return value }

func rtg_runtime_ArenaPersistBytes(value []byte) []byte { return value }

func RunRTGCommand(args []string, env []string) int {
	commandArgs := dropProgramArg(args)
	resetArena := rtgFrontendCanResetArena()
	mark := 0
	if resetArena {
		mark = rtg_runtime_ArenaMark()
	}
	built := BuildFromFS(commandArgs, rtgWorkDir(env), rtgStdRoot(args, env), RTGFS{})
	if !built.Ok {
		printRTGBuildError(built)
		return 1
	}
	unit := built.Unit
	target := built.Options.Target
	output := built.Options.Output
	strip := built.Options.Strip
	if resetArena {
		unit = rtg_runtime_ArenaPersistBytes(unit)
		target = rtg_runtime_ArenaPersistString(target)
		output = rtg_runtime_ArenaPersistString(output)
		rtg_runtime_ArenaReset(mark)
	}
	if !backendbridge.CompileUnitToOutputStripEnv(unit, target, output, strip, args, env) {
		print("rtg: backend compilation failed\n")
		return 1
	}
	return 0
}

func dropProgramArg(args []string) []string {
	if len(args) == 0 {
		return args
	}
	return args[1:]
}

func rtgWorkDir(env []string) string {
	for i := 0; i < len(env); i++ {
		item := env[i]
		if len(item) >= 4 && item[0] == 'P' && item[1] == 'W' && item[2] == 'D' && item[3] == '=' {
			return item[4:]
		}
	}
	return "."
}

func rtgStdRoot(args []string, env []string) string {
	for i := 0; i < len(env); i++ {
		item := env[i]
		if len(item) >= 12 &&
			item[0] == 'R' && item[1] == 'T' && item[2] == 'G' && item[3] == '_' &&
			item[4] == 'S' && item[5] == 'T' && item[6] == 'D' && item[7] == 'R' &&
			item[8] == 'O' && item[9] == 'O' && item[10] == 'T' && item[11] == '=' {
			return item[12:]
		}
	}
	bundled := rtgBundledStdRoot(args)
	if bundled != "" {
		return bundled
	}
	return "/std"
}

func rtgBundledStdRoot(args []string) string {
	dir := rtgExecutableDir(args)
	if dir == "" {
		return ""
	}
	path := rtgJoinPath(dir, "std")
	if rtgPathExists(path) {
		return path
	}
	path = rtgJoinPath(rtgJoinPath(dir, ".."), "std")
	if rtgPathExists(path) {
		return path
	}
	path = rtgJoinPath(rtgJoinPath(rtgJoinPath(dir, ".."), "share"), "rtg")
	path = rtgJoinPath(path, "std")
	if rtgPathExists(path) {
		return path
	}
	return ""
}

func rtgExecutableDir(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return rtgDirPath(args[0])
}

func rtgDirPath(path string) string {
	last := -1
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			last = i
		}
	}
	if last < 0 {
		return "."
	}
	if last == 0 {
		return "/"
	}
	return path[:last]
}

func rtgJoinPath(base string, elem string) string {
	if base == "" || base == "." {
		return elem
	}
	if elem == "" {
		return base
	}
	if base[len(base)-1] == '/' {
		return base + elem
	}
	return base + "/" + elem
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
	fd := open(rtgPathCString(path), 0)
	if fd < 0 {
		return nil, false
	}
	out := make([]byte, 0, 524288)
	buf := make([]byte, 4096)
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
	fd := open(rtgPathCString(path), 0)
	if fd < 0 {
		return nil, false
	}
	buf := make([]byte, 32768)
	out := make([]DirEntry, 0, 32)
	for {
		n := syscall(rtgGetdents64LinuxAmd64, fd, buf, len(buf))
		if n < 0 {
			n = syscall(rtgGetdents64LinuxAarch64, fd, buf, len(buf))
		}
		if n < 0 {
			n = syscall(rtgGetdents64Linux386, fd, buf, len(buf))
		}
		if n < 0 {
			close(fd)
			return nil, false
		}
		if n == 0 {
			break
		}
		pos := 0
		minimum := rtgDirentMinimum()
		for pos+minimum <= n {
			reclen := rtgDirentRecordLength(buf, pos)
			if reclen <= minimum || pos+reclen > n {
				close(fd)
				return nil, false
			}
			nameStart := rtgDirentNameStart(pos)
			typeAt := rtgDirentTypeOffset(pos)
			nameEnd := nameStart
			for nameEnd < pos+reclen && buf[nameEnd] != 0 {
				nameEnd++
			}
			if nameEnd > nameStart && !rtgDirNameIsDot(buf, nameStart, nameEnd) {
				out = append(out, DirEntry{Name: string(buf[nameStart:nameEnd]), IsDir: buf[typeAt] == 4})
			}
			pos += reclen
		}
	}
	close(fd)
	sortDirEntries(out)
	return out, true
}

func rtgDirNameIsDot(buf []byte, start int, end int) bool {
	if end-start == 1 && buf[start] == '.' {
		return true
	}
	if end-start == 2 && buf[start] == '.' && buf[start+1] == '.' {
		return true
	}
	return false
}

func rtgPathCString(path string) string {
	var out []byte
	for i := 0; i < len(path); i++ {
		out = append(out, path[i])
	}
	out = append(out, 0)
	return string(out)
}

func printRTGBuildError(result BuildResult) {
	if result.Error == BuildErrOptions {
		printOptionError(result.Options)
		return
	}
	if result.Error == BuildErrSource {
		if result.Sources.Error == SourceErrMissingModule {
			print("rtg: missing module at ")
		} else if result.Sources.Error == SourceErrModule {
			print("rtg: invalid module at ")
		} else if result.Sources.Error == SourceErrPackageArg {
			print("rtg: invalid package argument: ")
		} else if result.Sources.Error == SourceErrReadDir {
			print("rtg: failed to read directory: ")
		} else if result.Sources.Error == SourceErrReadFile {
			print("rtg: failed to read file: ")
		} else if result.Sources.Error == SourceErrParse {
			print("rtg: failed to parse source: ")
		} else if result.Sources.Error == SourceErrImport {
			print("rtg: failed to resolve import: ")
		} else {
			print("rtg: source error at ")
		}
		print(result.ErrorPath)
		print("\n")
		return
	}
	if result.Error == BuildErrPipeline {
		print("rtg: frontend pipeline failed: error=")
		rtgPrintInt(result.Pipeline.Error)
		print(" workspace=")
		rtgPrintInt(result.Pipeline.Workspace.Error)
		print(" graph=")
		rtgPrintInt(result.Pipeline.Workspace.Graph.Error)
		print(" graphPackage=")
		rtgPrintInt(result.Pipeline.Workspace.Graph.ErrorPackage)
		graphPackage := result.Pipeline.Workspace.Graph.ErrorPackage
		if graphPackage >= 0 && graphPackage < len(result.Pipeline.Workspace.Graph.Packages) {
			graphPkg := result.Pipeline.Workspace.Graph.Packages[graphPackage]
			print(" packageError=")
			rtgPrintInt(graphPkg.Error)
			print(" packageFile=")
			rtgPrintInt(graphPkg.ErrorFile)
			print(" packageImport=")
			rtgPrintInt(graphPkg.ErrorImport)
			if graphPkg.ErrorFile >= 0 && graphPkg.ErrorFile < len(graphPkg.Files) {
				graphFile := graphPkg.Files[graphPkg.ErrorFile]
				print(" packagePath=")
				print(graphFile.Path)
				print(" parseError=")
				rtgPrintInt(graphFile.File.Error)
				print(" parseToken=")
				rtgPrintInt(graphFile.File.ErrorTok)
			}
		}
		print(" package=")
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

//go:build rtg

package driver

import rtgx "j5.nz/rtg"

const rtgGetdents64LinuxAmd64 = 217

type RTGFS struct{}

func syscall(num int, fd int, buf []byte, size int) int { return 0 }

func RunRTGCommand(args []string, env []string) int {
	commandArgs := dropProgramArg(args)
	built := BuildFromFS(commandArgs, rtgWorkDir(env), "/std", RTGFS{})
	if !built.Ok {
		printRTGBuildError(built)
		return 1
	}
	if !rtgx.RtgCompileUnitToOutputStrip(built.Unit, built.Options.Target, built.Options.Output, built.Options.Strip) {
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

func (fs RTGFS) ReadFile(path string) ([]byte, bool) {
	fd := open(rtgPathCString(path), 0)
	if fd < 0 {
		return nil, false
	}
	var out []byte
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
	var out []DirEntry
	for {
		n := syscall(rtgGetdents64LinuxAmd64, fd, buf, len(buf))
		if n < 0 {
			close(fd)
			return nil, false
		}
		if n == 0 {
			break
		}
		pos := 0
		for pos+19 <= n {
			reclen := int(buf[pos+16]) | int(buf[pos+17])<<8
			if reclen <= 19 || pos+reclen > n {
				close(fd)
				return nil, false
			}
			nameStart := pos + 19
			nameEnd := nameStart
			for nameEnd < pos+reclen && buf[nameEnd] != 0 {
				nameEnd++
			}
			if nameEnd > nameStart && !rtgDirNameIsDot(buf, nameStart, nameEnd) {
				out = append(out, DirEntry{Name: string(buf[nameStart:nameEnd]), IsDir: buf[pos+18] == 4})
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
		print("rtg: frontend pipeline failed\n")
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

package driver

import (
	"path"
	"renvo.dev/internal/elflink"
	"strings"
)

// This table maps actual implemented libc exports to their source objects.
// No symbol is claimed until its object has compiled and the linker resolves it.
func libcObjectForSymbol(symbol string) string {
	switch symbol {
	case "mbsinit", "mbrtowc", "mbrlen", "wcrtomb", "mbsrtowcs", "wcsrtombs", "wcslen", "wcscmp", "wcscpy", "fputwc", "putwc", "putwchar", "fputws", "vfwprintf", "fwprintf", "vwprintf", "wprintf":
		return "wchar.c"
	case "setlocale", "__renvo_locale_utf8":
		return "locale.c"
	case "malloc", "calloc", "realloc", "free", "strtoul", "strtol", "atoi", "atol", "atoll", "abs", "labs", "abort", "atexit", "_Exit", "exit", "__renvo_libc_finish", "getenv", "__renvo_mb_cur_max":
		return "stdlib.c"
	case "strstr", "strnlen", "memchr", "strcspn", "strspn", "memcpy", "memmove", "memset", "memcmp", "strlen", "strcpy", "strncpy", "strcat", "strcmp", "strncmp", "strchr", "strrchr", "strerror", "strndup", "strdup":
		return "string.c"
	case "fopen", "fclose", "fflush", "__fpending", "__renvo_stream_error", "feof", "ferror", "clearerr", "fileno", "fputc", "fgetc", "ungetc", "getc", "putc", "putchar", "getchar", "fgets", "fputs", "fwrite", "fread", "puts", "vfprintf", "vsnprintf", "vsprintf", "snprintf", "sprintf", "vprintf", "printf", "fprintf", "stdin", "stdout", "stderr":
		return "stdio.c"
	case "isdigit", "islower", "isupper", "isalpha", "isalnum", "isblank", "iscntrl", "isgraph", "isprint", "ispunct", "isspace", "isxdigit", "tolower", "toupper":
		return "ctype.c"
	case "__renvo_assert_fail":
		return "assert.c"
	case "_exit", "dup2", "close":
		return "unistd.c"
	case "__renvo_libc_start", "__renvo_c_getenv", "__renvo_c_abort", "__renvo_c_close", "__renvo_c_dup2", "__renvo_c_open", "__renvo_c_write_byte", "__renvo_c_read_byte", "program_invocation_name", "program_invocation_short_name":
		return "linux_object_runtime.c"
	case "__renvo_linux_syscall":
		return "syscall_bridge.go"

	case "getrlimit":
		return "resource.c"
	case "__renvo_c_getrlimit":
		return "resource_bridge.go"
	case "fcntl":
		return "fcntl.c"
	case "errno":
		return "errno.c"
	case "__renvo_c_fcntl":
		return "fcntl_bridge.go"
	}
	return ""
}

// objectSourceFS shadows only the generated library source, without borrowing
// the caller's C translation unit or changing its declarations/macros.
type objectSourceFS struct {
	SourceFS
	name   string
	source []byte
}

func (f objectSourceFS) ReadFile(name string) ([]byte, bool) {
	if path.Clean(name) == f.name {
		return f.source, true
	}
	return f.SourceFS.ReadFile(name)
}
func (f objectSourceFS) PathExists(name string) bool {
	if path.Clean(name) == f.name {
		return true
	}
	return f.SourceFS.PathExists(name)
}
func (f objectSourceFS) ReadDir(name string) ([]DirEntry, bool) {
	if path.Clean(name) == path.Dir(f.name) {
		return []DirEntry{{Name: path.Base(f.name)}}, true
	}
	return f.SourceFS.ReadDir(name)
}

const syscallObjectBridge = `package main
func syscall(number int,a uintptr,b uintptr,c uintptr) int{return 0}
//export __renvo_linux_syscall
func __renvo_linux_syscall(number int64,a uint64,b uint64,c uint64) int64 {return int64(syscall(int(number),uintptr(a),uintptr(b),uintptr(c)))}
`
const resourceObjectBridge = `package main
import "unsafe"
func syscall(number int,a uintptr,b uintptr,c uintptr) int{return 0}
//export __renvo_c_getrlimit
func __renvo_c_getrlimit(resource int32, limit *byte) int32 {return int32(syscall(97,uintptr(resource),uintptr(unsafe.Pointer(limit)),0))}
`
const fcntlObjectBridge = `package main
func syscall(number int,a uintptr,b uintptr,c uintptr) int{return 0}
//export __renvo_c_fcntl
func __renvo_c_fcntl(fd int32,action int32,arg int32) int32 {return int32(syscall(72,uintptr(fd),uintptr(action),uintptr(arg)))}
`

func resolveLibcObjects(inputs []elflink.Input, fs SourceFS) ([]elflink.Input, *CommandResult, error) {
	loaded := make(map[string]bool)
	for {
		unresolved, err := elflink.Unresolved(inputs)
		if err.Message != "" {
			return nil, linkFailure(err.Input, err.Message), nil
		}
		progress := false
		for _, symbol := range unresolved {
			name := libcObjectForSymbol(symbol)
			if name == "" || loaded[name] {
				continue
			}
			loaded[name] = true
			var source []byte
			if name == "syscall_bridge.go" {
				source = []byte(syscallObjectBridge)
			} else if name == "resource_bridge.go" {
				source = []byte(resourceObjectBridge)
			} else if name == "fcntl_bridge.go" {
				source = []byte(fcntlObjectBridge)
			} else {
				var ok bool
				source, ok = fs.ReadFile("libc/src/" + name)
				if !ok {
					return nil, linkFailure(name, "could not read libc source"), nil
				}
			}
			virtual := "__renvo_link_libc/" + name
			libraryFS := objectSourceFS{SourceFS: fs, name: virtual, source: source}
			args := []string{"-c", virtual, "-o", "library.o"}
			if strings.HasSuffix(name, ".c") {
				args = append([]string{"cc"}, args...)
			}
			result, e := CompileCommand(&CommandRequest{Filesystem: libraryFS, Args: args, Target: "linux/amd64"})
			if e != nil {
				return nil, nil, e
			}
			if !result.Ok {
				return nil, result, nil
			}
			inputs = append(inputs, elflink.Input{Name: "libc/" + name + ".o", Data: result.Binary})
			progress = true
		}
		if !progress {
			return inputs, nil, nil
		}
	}
}

// A hosted source compilation may reference a separately declared libc symbol
// without including its header. Compile it as its own object and resolve real
// library definitions, keeping incompatible probe prototypes in separate units.
func linkCSourceObject(args []string, output string, fs SourceFS) (*CommandResult, error) {
	objectArgs := make([]string, 0, len(args)+3)
	for i := 1; i < len(args); i++ {
		if args[i] == "-o" {
			i++
			continue
		}
		objectArgs = append(objectArgs, args[i])
	}
	objectArgs = append(objectArgs, "-c", "-o", "source.o")
	object, e := CompileCommand(&CommandRequest{Filesystem: fs, Args: objectArgs, Target: "linux/amd64"})
	if e != nil || !object.Ok {
		return object, e
	}
	inputs, failed, e := resolveLibcObjects([]elflink.Input{{Name: "source.o", Data: object.Binary}}, fs)
	if e != nil || failed != nil {
		return failed, e
	}
	image, err := elflink.Link(inputs)
	if err.Message != "" {
		return linkFailure(err.Input, err.Message), nil
	}
	result := commandResult(true, Diagnostic{}, output, image, "", nil)
	for name, data := range object.Outputs {
		if name != object.Output {
			result.Outputs[name] = data
		}
	}
	return result, nil
}

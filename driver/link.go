package driver

import (
	"bytes"
	"fmt"
	"strings"

	internal "renvo.dev/internal/driver"
	"renvo.dev/internal/elflink"
)

// LinkerVersion identifies Renvo's static object linker, not GNU ld or BFD.
const LinkerVersion = "renvo ld (static Linux/amd64 object linker)\n"

// LinkCommand links virtual ELF64/amd64 relocatable objects using Renvo's
// main-calling startup. It does not supply libc, search host libraries, launch
// processes, or write files. Args omit the executable name. Supported options
// are -o FILE, -static, -s, -nostdlib, -v, --version and --, plus @response files.
// The output is already stripped. Other options (including shared/relocatable
// output, entry overrides and linker scripts) are not supported.
// Bad command inputs are diagnostics; invalid API requests are Go errors.
func LinkCommand(request *CommandRequest) (*CommandResult, error) {
	return linkCommandLibraries(request, false)
}
func linkCommandLibraries(request *CommandRequest, libc bool) (*CommandResult, error) {
	if request == nil || request.Filesystem == nil {
		return nil, fmt.Errorf("source filesystem must be specified")
	}
	if request.Target != "" && request.Target != "linux/amd64" {
		return linkFailure("", "object linking only supports linux/amd64"), nil
	}
	expanded := internal.ExpandCCompilerResponseFiles(append([]string{"renvo", "cc"}, request.Args...), request.Filesystem)
	if !expanded.Ok {
		return linkFailure(expanded.ErrorPath, "could not expand response file"), nil
	}
	args := expanded.Args[2:]
	output := "a.out"
	var names []string
	version, endOptions := false, false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !endOptions {
			switch arg {
			case "--":
				endOptions = true
				continue
			case "-v", "--version":
				version = true
				continue
			case "-nostdlib":
				libc = false
				continue
			case "-static", "-s":
				continue
			case "-o":
				i++
				if i == len(args) || args[i] == "" {
					return linkFailure("", "-o requires an output filename"), nil
				}
				output = args[i]
				continue
			}
			if strings.HasPrefix(arg, "-") {
				return linkFailure("", "unsupported linker option "+arg), nil
			}
		}
		names = append(names, arg)
	}
	if len(names) == 0 && version {
		return commandResult(true, Diagnostic{}, "-", []byte(LinkerVersion), "", nil), nil
	}
	// '-' is the result protocol's stdout channel, not a virtual filename.
	if output == "-" {
		return linkFailure("", "linker output to stdout is unsupported"), nil
	}
	var inputs []elflink.Input
	for _, name := range names {
		data, ok := request.Filesystem.ReadFile(name)
		if !ok {
			return linkFailure(name, "could not read object"), nil
		}
		if bytes.HasPrefix(data, []byte("!<arch>\n")) {
			var e elflink.Error
			inputs, e = elflink.ExtractArchive(inputs, elflink.Input{Name: name, Data: data})
			if e.Message != "" {
				return linkFailure(e.Input, e.Message), nil
			}
		} else {
			inputs = append(inputs, elflink.Input{Name: name, Data: data})
		}
	}
	if libc {
		var failed *CommandResult
		var e error
		inputs, failed, e = resolveLibcObjects(inputs, request.Filesystem)
		if failed != nil || e != nil {
			return failed, e
		}
	}
	image, err := elflink.Link(inputs)
	if err.Message != "" {
		return linkFailure(err.Input, err.Message), nil
	}
	result := commandResult(true, Diagnostic{}, output, image, "", nil)
	if version {
		result.Outputs["-"] = []byte(LinkerVersion)
	}
	return result, nil
}
func linkFailure(path, message string) *CommandResult {
	return commandResult(false, Diagnostic{Phase: "linker", Code: "RENVO-LINK-001", Path: path, Message: message}, "", nil, "", nil)
}

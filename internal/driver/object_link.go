package driver

import "renvo.dev/internal/elflink"

func ObjectLinkCommandRequested(args []string) bool {
	if len(args) < 3 || args[1] != "cc" {
		return false
	}
	found := false
	for i := 2; i < len(args); i++ {
		arg := args[i]
		if arg == "-c" || arg == "-E" || arg == "-S" {
			return false
		}
		if arg == "-o" || arg == "-t" {
			i++
			continue
		}
		if len(arg) > 2 && (arg[len(arg)-2:] == ".c" || arg[len(arg)-2:] == ".i") {
			return false
		}
		if len(arg) > 2 && arg[len(arg)-2:] == ".o" {
			found = true
		}
	}
	return found
}

func LinkObjectCommand(args []string, fs SourceFS) (string, []byte, string) {
	output := "a.out"
	var inputs []elflink.Input
	for i := 2; i < len(args); i++ {
		arg := args[i]
		if arg == "-o" || arg == "-t" {
			if i+1 == len(args) {
				return "", nil, "renvo cc: missing value for " + arg + "\n"
			}
			i++
			if arg == "-o" {
				output = args[i]
			} else if args[i] != "linux/amd64" {
				return "", nil, "renvo cc: object linking only supports linux/amd64\n"
			}
			continue
		}
		if arg == "-s" || arg == "-static" || arg == "-nostdlib" {
			continue
		}
		if len(arg) > 0 && arg[0] == '-' {
			return "", nil, "renvo cc: unsupported linker option " + arg + "\n"
		}
		data, ok := fs.ReadFile(arg)
		if !ok {
			return "", nil, "renvo cc: could not read object " + arg + "\n"
		}
		inputs = append(inputs, elflink.Input{Name: arg, Data: data})
	}
	image, err := elflink.Link(inputs)
	if err.Message != "" {
		return "", nil, err.Input + ": error RENVO-LINK-001 (linker): " + err.Message + "\n"
	}
	return output, image, ""
}

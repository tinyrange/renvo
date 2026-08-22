package driver

import "renvo.dev/internal/c11"

const (
	CCompilerRequestNone = iota
	CCompilerRequestVersion
	CCompilerRequestDumpVersion
	CCompilerRequestDumpMachine
	CCompilerRequestPreprocessStdin
	CCompilerRequestAssemblerVersion
)

const CCompilerVersionText = "renvo cc (GCC-compatible) 5.1.0\n"

type CCompilerRequest struct {
	Kind int
}

type CCompilerResponseResult struct {
	Args      []string
	Ok        bool
	ErrorPath string
}

// ExpandCCompilerResponseFiles expands GCC-style @files before option
// classification. Quoting and backslash escapes are handled without invoking
// a shell, and recursion is bounded so malformed build inputs fail promptly.
func ExpandCCompilerResponseFiles(args []string, fs SourceFS) CCompilerResponseResult {
	if len(args) < 2 || args[1] != "cc" {
		return CCompilerResponseResult{Args: args, Ok: true}
	}
	out := make([]string, 2)
	out[0] = args[0]
	out[1] = args[1]
	for i := 2; i < len(args); i++ {
		if !expandCCompilerArgument(&out, args[i], fs, 0) {
			path := args[i]
			if len(path) > 0 && path[0] == '@' {
				path = path[1:]
			}
			return CCompilerResponseResult{Ok: false, ErrorPath: path}
		}
	}
	return CCompilerResponseResult{Args: out, Ok: true}
}

func expandCCompilerArgument(out *[]string, arg string, fs SourceFS, depth int) bool {
	if len(arg) < 2 || arg[0] != '@' {
		*out = append(*out, arg)
		return true
	}
	if depth >= 8 || fs == nil {
		return false
	}
	source, ok := fs.ReadFile(arg[1:])
	if !ok {
		return false
	}
	words, ok := splitCCompilerResponse(source)
	if !ok {
		return false
	}
	for i := 0; i < len(words); i++ {
		if !expandCCompilerArgument(out, words[i], fs, depth+1) {
			return false
		}
	}
	return true
}

func splitCCompilerResponse(source []byte) ([]string, bool) {
	var words []string
	var word []byte
	quote := byte(0)
	escaped := false
	started := false
	for i := 0; i <= len(source); i++ {
		if i == len(source) && escaped {
			return nil, false
		}
		ch := byte(' ')
		if i < len(source) {
			ch = source[i]
		}
		if escaped {
			word = append(word, ch)
			escaped = false
			started = true
			continue
		}
		if ch == '\\' {
			escaped = true
			started = true
			continue
		}
		if quote != 0 {
			if ch == quote {
				quote = 0
			} else if i == len(source) {
				return nil, false
			} else {
				word = append(word, ch)
			}
			started = true
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			started = true
			continue
		}
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' || i == len(source) {
			if started {
				words = append(words, string(word))
				word = nil
				started = false
			}
			continue
		}
		word = append(word, ch)
		started = true
	}
	return words, !escaped && quote == 0
}

// InspectCCompilerRequest recognizes compiler metadata operations before the
// package-oriented build path. Other invocations continue through the strict
// option classifier and cannot accidentally succeed as feature probes.
func InspectCCompilerRequest(args []string) CCompilerRequest {
	if len(args) < 3 || args[1] != "cc" {
		return CCompilerRequest{}
	}
	if len(args) == 3 {
		switch args[2] {
		case "--version":
			return CCompilerRequest{Kind: CCompilerRequestVersion}
		case "-dumpversion", "-dumpfullversion":
			return CCompilerRequest{Kind: CCompilerRequestDumpVersion}
		case "-dumpmachine":
			return CCompilerRequest{Kind: CCompilerRequestDumpMachine}
		}
	}
	preprocess := false
	stdin := false
	languageC := false
	languageAssembler := false
	assemblerVersion := false
	validPreprocess := true
	validAssembler := true
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "-E":
			preprocess = true
			validAssembler = false
		case "-P":
			validAssembler = false
		case "-":
			stdin = true
			validAssembler = false
		case "-Wa,--version":
			assemblerVersion = true
			validPreprocess = false
		case "-c", "/dev/null":
			validPreprocess = false
		case "-o":
			validPreprocess = false
			if i+1 < len(args) && args[i+1] == "/dev/null" {
				i++
			} else {
				validAssembler = false
			}
		case "-x":
			if i+1 < len(args) {
				i++
				languageC = args[i] == "c"
				languageAssembler = args[i] == "assembler-with-cpp"
				if !languageC {
					validPreprocess = false
				}
				if !languageAssembler {
					validAssembler = false
				}
			} else {
				validPreprocess = false
				validAssembler = false
			}
		default:
			validPreprocess = false
			validAssembler = false
		}
	}
	if assemblerVersion && languageAssembler && validAssembler {
		return CCompilerRequest{Kind: CCompilerRequestAssemblerVersion}
	}
	if preprocess && stdin && languageC && validPreprocess {
		return CCompilerRequest{Kind: CCompilerRequestPreprocessStdin}
	}
	return CCompilerRequest{}
}

func ExecuteCCompilerRequest(request CCompilerRequest, input []byte) (int, string) {
	switch request.Kind {
	case CCompilerRequestVersion:
		return 0, CCompilerVersionText
	case CCompilerRequestDumpVersion:
		return 0, "5.1.0\n"
	case CCompilerRequestDumpMachine:
		return 0, "x86_64-linux-gnu\n"
	case CCompilerRequestPreprocessStdin:
		result := c11.PreprocessProbe(input, []c11.Macro{
			{Name: "__RENVO__", Value: "1"},
			{Name: "__GNUC__", Value: "5"},
			{Name: "__GNUC_MINOR__", Value: "1"},
			{Name: "__GNUC_PATCHLEVEL__", Value: "0"},
			{Name: "__x86_64__", Value: "1"},
			{Name: "__linux__", Value: "1"},
		})
		if !result.Ok {
			return 1, "renvo cc: unsupported preprocessor directive in compiler probe\n"
		}
		return 0, string(result.Source)
	case CCompilerRequestAssemblerVersion:
		// Linux 6.12 requires GNU binutils 2.25. The standalone assembler
		// remains an external tool; this is the minimum bridge contract that
		// Renvo advertises while instruction probes themselves stay strict.
		return 0, "GNU assembler (GNU Binutils) 2.25\n"
	}
	return 1, ""
}

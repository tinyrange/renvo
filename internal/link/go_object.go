package link

import (
	"renvo.dev/internal/syntax"
	"renvo.dev/internal/unit"
)

// Go object roots expose package functions through the existing object ABI.
// main uses the generated entrypoint so package initialization runs first and
// the C entrypoint receives an integer status, not Go main's void result.
func goObjectExportsCore(program unit.Program) []string {
	exports := make([]string, len(program.Tokens))
	explicitMain := false
	for _, fn := range program.Funcs {
		if goObjectExportDirective(program.Text, program.Tokens[fn.StartTok].Start) == "main" {
			explicitMain = true
		}
	}
	for _, fn := range program.Funcs {
		if fn.ReceiverEnd > fn.ReceiverStart || fn.BodyEnd <= fn.BodyStart {
			continue
		}
		name := coreLinkedProgramText(program, fn.NameStart, fn.NameEnd)
		if name == "appMain" && program.Package == "main" {
			if explicitMain {
				continue
			}
			name = "main"
		} else if !syntax.IdentifierExported(program.Text, fn.NameStart) {
			continue
		}
		if goObjectExportDirective(program.Text, program.Tokens[fn.StartTok].Start) != "" {
			continue
		}
		exports[fn.StartTok] = name
	}
	return exports
}

func goObjectExportDirective(text []byte, before int) string {
	// Directives belong to the contiguous comment group before a declaration.
	end := before
	for end > 0 {
		for end > 0 && (text[end-1] == ' ' || text[end-1] == '\t' || text[end-1] == '\r' || text[end-1] == '\n') {
			end--
		}
		start := end
		for start > 0 && text[start-1] != '\n' {
			start--
		}
		lineStart := start
		for lineStart < end && (text[lineStart] == ' ' || text[lineStart] == '\t') {
			lineStart++
		}
		line := coreText(text, lineStart, end)
		if len(line) < 2 || line[:2] != "//" {
			return ""
		}
		if len(line) > 9 && line[:9] == "//export " {
			return line[9:]
		}
		end = start
	}
	return ""
}

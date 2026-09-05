package link

import "renvo.dev/internal/arena"
import "renvo.dev/internal/unit"

// Capture linked names before transient package storage is retired. The
// dependency graph, not package spelling guesses, identifies the handler.
func coreDefaultHandlerNames(programs []unit.Program, aliases []string, offsets []int) []string {
	names := make([]string, 3)
	for i := 0; i < len(programs); i++ {
		path := programs[i].ImportPath
		if path != "renvo.dev/x/runtime" && path != "renvo.dev/x/runtime/serial" {
			continue
		}
		for j := 0; j < len(programs[i].Symbols); j++ {
			name := programs[i].Symbols[j].Name
			index := -1
			if path == "renvo.dev/x/runtime" && name == "requireHandler" {
				index = 0
			}
			if path == "renvo.dev/x/runtime" && name == "activeHandler" {
				index = 1
			}
			if path == "renvo.dev/x/runtime/serial" && name == "New" {
				index = 2
			}
			if index >= 0 {
				alias := corePackageSymbolAlias(aliases, offsets, i, j)
				if alias == "" {
					alias = name
				}
				names[index] = cloneCoreLinkString(alias)
			}
		}
	}
	return names
}

// Select the target's default serialized handler on the first operation. An
// explicitly installed handler takes precedence, including during package
// initialization. Backend units still contain only ordinary source semantics.
func lowerDefaultHandler(program *unit.Program, names []string, transient bool) bool {
	if names[0] == "" || names[1] == "" || names[2] == "" {
		return true
	}
	index := findCoreFuncByName(*program, names[0])
	if index < 0 {
		return false
	}
	fn := program.Funcs[index]
	close := functionValueFindMatchingBrace(program, fn.BodyStart)
	if close < 0 {
		return false
	}
	replacement := "{ if " + names[1] + " == nil { " + names[1] + " = " + names[2] + "() }; return " + names[1] + " }"
	edits := []functionValueEdit{functionValueTokenRangeEdit(program, fn.BodyStart, close+1, replacement)}
	edits = appendFunctionValuePackageEdits(program, edits)
	originalLength := len(program.Text)
	if transient {
		renvo_runtime_ArenaDiscardLinkTokens(program.Tokens)
	}
	text, ok := applyFunctionValueEdits(program.Text, edits)
	if transient {
		arena.DiscardBytes(program.Text)
	}
	if !ok {
		return false
	}
	return reparseFunctionValueProgram(program, text, edits, originalLength, len(text))
}

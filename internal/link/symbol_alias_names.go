package link

import "renvo.dev/internal/unit"

// Linked packages share one namespace. A generated name must also avoid local
// bindings: replacing p.Value with Renvop0_Value would otherwise resolve to a
// local variable with that spelling. Reserve authored identifiers throughout
// the graph before choosing names, including names containing suffixes.
func coreAvoidSymbolAliasCollisions(programs []unit.Program, offsets []int, aliases []string) {
	var authored []string
	for i := range programs {
		program := &programs[i]
		for _, symbol := range program.Symbols {
			if functionValueHasPrefix(symbol.Name, "Renvop") || functionValueHasPrefix(symbol.Name, "renvop") || functionValueHasPrefix(symbol.Name, "renvoi") || functionValueHasPrefix(symbol.Name, "renvo_runtime_M") || functionValueHasPrefix(symbol.Name, "renvo_runtime_FmtPrintln") || functionValueHasPrefix(symbol.Name, "renvo_runtime_Syscall") {
				authored = append(authored, symbol.Name)
			}
		}
		for _, token := range program.Tokens {
			if token.KindLine&255 == unit.TokenIdent && coreGeneratedAliasText(program.Text, token.Start, token.Start+token.Size) {
				authored = append(authored, coreText(program.Text, token.Start, token.Start+token.Size))
			}
		}
	}
	// Only identifiers in the generated namespaces can collide. Filtering before
	// copying token text keeps this table small even when linking the compiler.
	names := make([]string, (len(authored)+len(aliases))*2+1)
	for _, name := range authored {
		coreReserveAliasName(names, name)
	}
	// A declaration directive carries intrinsic identity when its usual alias
	// is occupied. Typed syscall declarations still share one chosen alias.
	var intrinsicAliases []string
	for i := range programs {
		for j, symbol := range programs[i].Symbols {
			index := offsets[i] + j
			name := aliases[index]
			if name == "" {
				continue
			}
			intrinsic := coreCompilerIntrinsicAlias(programs[i].ImportPath, symbol.Name)
			if intrinsic != "" {
				shared := ""
				for k := 0; k < len(intrinsicAliases); k += 2 {
					if intrinsicAliases[k] == intrinsic {
						shared = intrinsicAliases[k+1]
					}
				}
				if shared != "" {
					aliases[index] = shared
					continue
				}
			}
			for !coreReserveAliasName(names, name) {
				name += "_"
			}
			aliases[index] = name
			if intrinsic != "" {
				intrinsicAliases = append(intrinsicAliases, intrinsic, name)
			}
		}
	}
}

func coreGeneratedAliasText(text []byte, start, end int) bool {
	if start < 0 || end > len(text) || end-start < 6 {
		return false
	}
	first := text[start]
	if (first != 'r' && first != 'R') || text[start+1] != 'e' || text[start+2] != 'n' || text[start+3] != 'v' || text[start+4] != 'o' {
		return false
	}
	if text[start+5] == 'p' {
		return true
	}
	if first != 'r' {
		return false
	}
	if text[start+5] == 'i' {
		return true
	}
	prefix := "renvo_runtime_M"
	if end-start > 14 && text[start+14] == 'F' {
		prefix = "renvo_runtime_FmtPrintln"
	} else if end-start > 14 && text[start+14] == 'S' {
		prefix = "renvo_runtime_Syscall"
	}
	if end-start < len(prefix) {
		return false
	}
	for i := 5; i < len(prefix); i++ {
		if text[start+i] != prefix[i] {
			return false
		}
	}
	return true
}

// Return false when the spelling was already reserved. The table has room for
// every authored name and each final alias, so probing always finds a slot.
func coreReserveAliasName(names []string, name string) bool {
	bucket := coreSymbolAliasHash(name) % len(names)
	for names[bucket] != "" {
		if names[bucket] == name {
			return false
		}
		bucket++
		if bucket == len(names) {
			bucket = 0
		}
	}
	names[bucket] = name
	return true
}

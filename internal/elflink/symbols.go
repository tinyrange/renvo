package elflink

// Unresolved returns strong undefined symbols not satisfied by any input.
// Objects are parsed and validated without linking or writing intermediate files.
func Unresolved(inputs []Input) ([]string, Error) {
	var defined, needed []string
	for _, input := range inputs {
		obj, err := parseObject(input)
		if err.Message != "" {
			return nil, err
		}
		for _, table := range obj.symbols {
			for _, sym := range table {
				binding := sym.info >> 4
				if sym.name == "" || binding != stbGlobal && binding != stbWeak {
					continue
				}
				if sym.section != shnUndefined {
					defined = appendUniqueSymbol(defined, sym.name)
				} else if binding == stbGlobal {
					needed = appendUniqueSymbol(needed, sym.name)
				}
			}
		}
	}
	var unresolved []string
	for _, name := range needed {
		found := false
		for _, def := range defined {
			if name == def {
				found = true
				break
			}
		}
		if !found {
			unresolved = append(unresolved, name)
		}
	}
	return unresolved, Error{}
}
func appendUniqueSymbol(names []string, name string) []string {
	for _, old := range names {
		if old == name {
			return names
		}
	}
	return append(names, name)
}

package link

import "renvo.dev/internal/unit"

// Resolve aliases separately from defined types: aliases must not introduce
// duplicate dynamic type cases, while a defined slice keeps its own identity.
func reflectionCanonicalType(program *unit.Program, typ string, underlying bool) string {
	typ = functionValueCompactTypeText(typ)
	for depth := 0; depth <= len(program.Decls); depth++ {
		if typ == "byte" {
			return "uint8"
		}
		if typ == "rune" {
			return "int32"
		}
		next := ""
		for i := 0; i < len(program.Decls); i++ {
			decl := program.Decls[i]
			if decl.Kind != unit.TokenType {
				continue
			}
			name := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
			if functionValueTokenText(program, name) != typ {
				continue
			}
			start := name + 1
			alias := functionValueTokenEquals(program, start, "=")
			if !underlying && !alias {
				return typ
			}
			if alias {
				start++
			}
			end := functionValueTypeEnd(program, start)
			if end > start {
				next = functionValueCompactTypeText(functionValueTokensText(program, start, end))
			}
			break
		}
		if next == "" || next == typ {
			kind, element, key := reflectionCollectionParts(typ)
			if kind != "" {
				prefix := typ[:len(typ)-len(element)]
				element = reflectionCanonicalType(program, element, false)
				if kind == "pointer" {
					return "*" + element
				}
				if kind == "slice" {
					return "[]" + element
				}
				if kind == "map" {
					return "map[" + reflectionCanonicalType(program, key, false) + "]" + element
				}
				return prefix + element
			}
			return typ
		}
		typ = next
	}
	return typ
}

func reflectionCollectionParts(typ string) (string, string, string) {
	if len(typ) > 1 && typ[0] == '*' {
		return "pointer", typ[1:], ""
	}
	if functionValueHasPrefix(typ, "[]") {
		return "slice", typ[2:], ""
	}
	start := 0
	kind := "array"
	if functionValueHasPrefix(typ, "map[") {
		start = 3
		kind = "map"
	}
	if len(typ) <= start || typ[start] != '[' {
		return "", "", ""
	}
	depth := 1
	end := start + 1
	for end < len(typ) && depth > 0 {
		if typ[end] == '[' {
			depth++
		}
		if typ[end] == ']' {
			depth--
		}
		end++
	}
	if depth != 0 || end >= len(typ) {
		return "", "", ""
	}
	key := ""
	if kind == "map" {
		key = typ[start+1 : end-1]
	}
	return kind, typ[end:], key
}

func reflectionCollectionEdits(program *unit.Program, names coreReflectionNames, pending []string) ([]functionValueEdit, bool) {
	if names.collection == "" && names.rebuildCollection == "" && names.scalar == "" && names.scalarLike == "" && names.assign == "" {
		return nil, true
	}
	if names.collectionType == "" {
		return nil, false
	}
	read := "{ switch record := value.(type) {\n"
	build := "{ switch prototype.(type) {\n"
	scalar := "{ switch record := value.(type) {\n"
	scalarLike := "{ switch prototype.(type) {\n"
	assign := "{ switch record := target.(type) {\n"
	for _, typ := range []string{"bool", "string", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "float32", "float64", "any", "[]any", "map[string]any", "[]byte"} {
		pending = append(pending, typ, "*"+typ)
	}
	var seen []string
	for next := 0; next < len(pending); next++ {
		typ := reflectionCanonicalType(program, pending[next], false)
		if functionValueNameInList(seen, typ) {
			continue
		}
		seen = append(seen, typ)
		// Every registered non-pointer type also admits an addressable target.
		if len(typ) > 0 && typ[0] != '*' {
			pending = append(pending, "*"+typ)
		}
		underlying := reflectionCanonicalType(program, typ, true)
		kind, element, key := reflectionCollectionParts(underlying)
		if kind == "" {
			if reflectionScalarType(underlying) {
				scalar += "case " + typ + ": return " + underlying + "(record),true\n"
				scalarLike += "case " + typ + ": assigned,ok:=replacement.(" + underlying + "); if !ok { return nil,false }; return " + typ + "(assigned),true\n"
			}
			continue
		}
		pending = append(pending, element)
		if key != "" {
			pending = append(pending, key)
		}
		read += "case " + typ + ": var element " + element + "; out := " + names.collectionType + "{Kind:" + string(appendCoreQuotedString(nil, kind)) + ",Element:element}; "
		if kind != "array" {
			read += "out.Nil = record == nil; "
		}
		if kind == "pointer" {
			read += "if record != nil { out.Values=append(out.Values,*record) }; "
			assign += "case " + typ + ": if record==nil { return false }; "
			if element == "any" || element == "interface{}" {
				assign += "*record=replacement; return true\n"
			} else {
				assign += "assigned,ok:=replacement.(" + element + "); if !ok { return false }; *record=assigned; return true\n"
			}
		} else if kind == "map" {
			read += "var keyZero " + key + "; out.Key=keyZero; "
			read += "for key,item := range record { out.Keys=append(out.Keys,key); out.Values=append(out.Values,item) }; "
		} else {
			read += "for _,item := range record { out.Values=append(out.Values,item) }; "
		}
		read += "return out,true\n"
		build += "case " + typ + ": "
		if kind == "array" {
			build += "if nilValue { return nil,false }; var out " + typ + "; if len(keys)!=0 || len(values)!=len(out) { return nil,false }; "
		} else {
			build += "if nilValue { if len(keys)!=0 || len(values)!=0 { return nil,false }; var zero " + typ + "; return zero,true }; "
			if kind == "pointer" {
				build += "if len(keys)!=0 || len(values)!=1 { return nil,false }; "
				if element == "any" || element == "interface{}" {
					build += "item:=values[0]; "
				} else {
					build += "item,ok:=values[0].(" + element + "); if !ok { return nil,false }; "
				}
				build += "return (" + typ + ")(&item),true\n"
				continue
			}
			if kind == "slice" {
				build += "if len(keys)!=0 { return nil,false }; out:=make(" + typ + ",len(values)); "
			} else {
				build += "if len(keys)!=len(values) { return nil,false }; out:=make(" + typ + "); "
			}
		}
		build += "for i:=0; i<len(values); i++ { "
		if element == "any" || element == "interface{}" {
			build += "item:=values[i]; "
		} else {
			build += "item,ok:=values[i].(" + element + "); if !ok { return nil,false }; "
		}
		if kind == "map" {
			build += "key,ok:=keys[i].(" + key + "); if !ok { return nil,false }; out[key]=item"
		} else {
			build += "out[i]=item"
		}
		build += " }; return out,true\n"
	}
	read += "}; return " + names.collectionType + "{},false }"
	build += "}; return nil,false }"
	scalar += "}; return nil,false }"
	scalarLike += "}; return nil,false }"
	assign += "}; return false }"
	var edits []functionValueEdit
	for i := 0; i < 5; i++ {
		name := names.collection
		body := read
		if i == 1 {
			name = names.rebuildCollection
			body = build
		}
		if i == 2 {
			name = names.scalar
			body = scalar
		}
		if i == 3 {
			name = names.scalarLike
			body = scalarLike
		}
		if i == 4 {
			name = names.assign
			body = assign
		}
		if name == "" {
			continue
		}
		index := findCoreFuncByName(*program, name)
		if index < 0 {
			return nil, false
		}
		fn := program.Funcs[index]
		end := functionValueFindMatchingBrace(program, fn.BodyStart)
		if end < 0 {
			return nil, false
		}
		edits = append(edits, functionValueTokenRangeEdit(program, fn.BodyStart, end+1, body))
	}
	return edits, true
}

func reflectionScalarType(typ string) bool {
	switch typ {
	case "bool", "string", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "float32", "float64", "complex64", "complex128":
		return true
	}
	return false
}

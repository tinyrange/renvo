package link

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/check"
	"renvo.dev/internal/syntax"
	"renvo.dev/internal/unit"
)

type coreReflectionNames struct {
	describe          string
	structType        string
	fieldType         string
	fieldValue        string
	setField          string
	collection        string
	collectionType    string
	rebuildCollection string
	scalar            string
	scalarLike        string
	structCopy        string
	assign            string
	aliases           []string
	originals         []string
}

// Resolve the intrinsic by import identity before package namespacing and
// transient retirement. A user package merely named reflect is not intrinsic.
func reflectionNamesCore(programs []unit.Program, aliases []string, offsets []int) coreReflectionNames {
	var out coreReflectionNames
	for i := 0; i < len(programs); i++ {
		for j := 0; j < len(programs[i].Symbols); j++ {
			name := programs[i].Symbols[j].Name
			alias := corePackageSymbolAlias(aliases, offsets, i, j)
			if alias == "" {
				alias = name
			}
			if programs[i].ImportPath == "reflect" {
				if name == "Describe" {
					out.describe = cloneCoreLinkString(alias)
				}
				if name == "Struct" {
					out.structType = cloneCoreLinkString(alias)
				}
				if name == "Field" {
					out.fieldType = cloneCoreLinkString(alias)
				}
				if name == "FieldValue" {
					out.fieldValue = cloneCoreLinkString(alias)
				}
				if name == "SetField" {
					out.setField = cloneCoreLinkString(alias)
				}
				if name == "InspectCollection" {
					out.collection = cloneCoreLinkString(alias)
				}
				if name == "Collection" {
					out.collectionType = cloneCoreLinkString(alias)
				}
				if name == "RebuildCollection" {
					out.rebuildCollection = cloneCoreLinkString(alias)
				}
				if name == "Scalar" {
					out.scalar = cloneCoreLinkString(alias)
				}
				if name == "ScalarLike" {
					out.scalarLike = cloneCoreLinkString(alias)
				}
				if name == "StructCopy" {
					out.structCopy = cloneCoreLinkString(alias)
				}
				if name == "Assign" {
					out.assign = cloneCoreLinkString(alias)
				}
			}
		}
	}
	if out.describe == "" {
		return out
	}
	for i := 0; i < len(programs); i++ {
		for j := 0; j < len(programs[i].Symbols); j++ {
			name := programs[i].Symbols[j].Name
			alias := corePackageSymbolAlias(aliases, offsets, i, j)
			if alias != "" && alias != name {
				out.aliases = append(out.aliases, cloneCoreLinkString(alias))
				out.originals = append(out.originals, cloneCoreLinkString(name))
			}
		}
	}
	return out
}

func reflectionOriginalName(names coreReflectionNames, name string) string {
	for i := 0; i < len(names.aliases); i++ {
		if names.aliases[i] == name {
			return names.originals[i]
		}
	}
	return name
}

func lowerReflectionCore(program *unit.Program, names coreReflectionNames, transient bool) bool {
	if names.describe == "" {
		return true
	}
	if names.structType == "" || names.fieldType == "" {
		return false
	}
	fnIndex := findCoreFuncByName(*program, names.describe)
	if fnIndex < 0 {
		return false
	}
	fn := program.Funcs[fnIndex]
	close := functionValueFindMatchingBrace(program, fn.BodyStart)
	if close < 0 {
		return false
	}
	packageEdits := appendFunctionValuePackageEdits(program, nil)
	source, ok := applyFunctionValueEdits(program.Text, packageEdits)
	if !ok {
		return false
	}
	file := syntax.ParseLinkedFile(source)
	if !file.Ok {
		return false
	}
	body := "{ switch value.(type) {\n"
	readBody := "{ switch record := value.(type) {\n"
	writeBody := "{ switch record := value.(type) {\n"
	copyBody := "{ switch record := value.(type) {\n"
	var collectionTypes []string
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if !syntax.ReflectDirective(file, decl) {
			continue
		}
		start := decl.NameTok + 1
		// Only named struct definitions opt in, not aliases or other types.
		if start+1 >= len(file.Tokens) || file.Tokens[start].KindLine&255 != syntax.TokenStruct {
			continue
		}
		open := start + 1
		end := open + 1
		depth := 1
		for end < len(file.Tokens) && depth > 0 {
			ch := file.Tokens[end].KindLine >> syntax.TokenOperatorCharShift & syntax.TokenOperatorCharMask
			if ch == '{' {
				depth++
			}
			if ch == '}' {
				depth--
			}
			end++
		}
		if depth != 0 {
			return false
		}
		nameToken := file.Tokens[decl.NameTok]
		typeName := string(file.Src[syntax.TokenStart(nameToken) : syntax.TokenStart(nameToken)+syntax.TokenSize(nameToken)])
		fields := check.StructFields(file, open+1, end-1)
		readCases := ""
		writeCases := ""
		fieldIndex := 0
		descriptor := names.structType + "{Name:" + string(appendCoreQuotedString(nil, reflectionOriginalName(names, typeName))) + ",Fields:[]" + names.fieldType + "{"
		for j := 0; j < len(fields); j++ {
			field := fields[j]
			fieldName := field.Name
			if field.NameTok < 0 {
				embedded := file.Tokens[field.TypeEnd-1]
				fieldName = reflectionOriginalName(names, string(file.Src[syntax.TokenStart(embedded):syntax.TokenStart(embedded)+syntax.TokenSize(embedded)]))
			}
			if !syntax.IdentifierExported([]byte(fieldName), 0) {
				continue
			}
			descriptor += "{Name:" + string(appendCoreQuotedString(nil, fieldName)) + ",Tag:" + string(appendCoreQuotedString(nil, field.Tag)) + "},"
			accessName := field.Name
			if field.NameTok < 0 {
				embedded := file.Tokens[field.TypeEnd-1]
				accessName = string(file.Src[syntax.TokenStart(embedded) : syntax.TokenStart(embedded)+syntax.TokenSize(embedded)])
			}
			first := file.Tokens[field.TypeStart]
			last := file.Tokens[field.TypeEnd-1]
			fieldType := string(file.Src[syntax.TokenStart(first) : syntax.TokenStart(last)+syntax.TokenSize(last)])
			collectionTypes = append(collectionTypes, fieldType)
			index := functionValueDecimal(fieldIndex)
			readCases += "case " + index + ": return record." + accessName + ",true\n"
			if compact := functionValueCompactTypeText(fieldType); compact == "any" || compact == "interface{}" {
				writeCases += "case " + index + ": record." + accessName + "=replacement; return true\n"
			} else {
				writeCases += "case " + index + ": assigned,ok := replacement.(" + fieldType + "); if !ok { return false }; record." + accessName + "=assigned; return true\n"
			}
			fieldIndex++
		}
		descriptor += "}}"
		body += "case " + typeName + ": return " + descriptor + ",true\n"
		body += "case *" + typeName + ": return " + descriptor + ",true\n"
		readBody += "case " + typeName + ": switch index {\n" + readCases + "}; return nil,false\n"
		readBody += "case *" + typeName + ": if record == nil { return nil,false }; switch index {\n" + readCases + "}; return nil,false\n"
		writeBody += "case *" + typeName + ": if record == nil { return false }; switch index {\n" + writeCases + "}; return false\n"
		copyBody += "case " + typeName + ": copied:=record; return &copied,true\n"
		copyBody += "case *" + typeName + ": var copied " + typeName + "; if record!=nil { copied=*record }; return &copied,true\n"
		collectionTypes = append(collectionTypes, "*"+typeName)
	}
	body += "}; return " + names.structType + "{},false }"
	readBody += "}; return nil,false }"
	writeBody += "}; return false }"
	copyBody += "}; return nil,false }"
	edits := []functionValueEdit{functionValueTokenRangeEdit(program, fn.BodyStart, close+1, body)}
	if names.fieldValue != "" {
		index := findCoreFuncByName(*program, names.fieldValue)
		if index < 0 {
			return false
		}
		getter := program.Funcs[index]
		end := functionValueFindMatchingBrace(program, getter.BodyStart)
		if end < 0 {
			return false
		}
		edits = append(edits, functionValueTokenRangeEdit(program, getter.BodyStart, end+1, readBody))
	}
	if names.setField != "" {
		index := findCoreFuncByName(*program, names.setField)
		if index < 0 {
			return false
		}
		setter := program.Funcs[index]
		end := functionValueFindMatchingBrace(program, setter.BodyStart)
		if end < 0 {
			return false
		}
		edits = append(edits, functionValueTokenRangeEdit(program, setter.BodyStart, end+1, writeBody))
	}
	collectionEdits, ok := reflectionCollectionEdits(program, names, collectionTypes)
	if !ok {
		return false
	}
	edits = append(edits, collectionEdits...)
	if names.structCopy != "" {
		index := findCoreFuncByName(*program, names.structCopy)
		if index < 0 {
			return false
		}
		fn := program.Funcs[index]
		end := functionValueFindMatchingBrace(program, fn.BodyStart)
		if end < 0 {
			return false
		}
		edits = append(edits, functionValueTokenRangeEdit(program, fn.BodyStart, end+1, copyBody))
	}
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

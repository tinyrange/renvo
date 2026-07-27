package rtg

import "renvo.dev/internal/syntax"

type embeddedFunction struct {
	Name       string
	Signature  []byte
	Parameters []string
	HasResult  bool
}

// appendArchitectureBindings specializes declarative instruction rows into
// ordinary direct Go wrappers. Neither the table nor an algorithm registry is
// retained in generated code.
func appendArchitectureBindings(out []byte, document Document, arch Declaration, rewriteAlgorithms bool) []byte {
	instructions, ok := declarationBlock(arch, "instructions")
	if !ok {
		return out
	}
	names := embeddedGoNames(document)
	prefix := "rtg" + exportedName(document.Unit)
	for i := 0; i < len(instructions.Children); i++ {
		left, right, assignment := statementAssignment(instructions.Children[i])
		if !assignment || len(left) != 1 || len(right) != 2 || right[0] != "go" {
			continue
		}
		function, found := findEmbeddedFunction(document, right[1])
		if !found {
			continue
		}
		out = append(out, "\n// Generated from instruction "...)
		out = append(out, left[0]...)
		out = append(out, ".\nfunc "...)
		out = append(out, prefix...)
		out = append(out, exportedName(left[0])...)
		if rewriteAlgorithms {
			out = appendRewrittenGo(out, function.Signature, names, prefix)
		} else {
			out = append(out, function.Signature...)
		}
		out = append(out, " {\n"...)
		if function.HasResult {
			out = append(out, "\treturn "...)
		} else {
			out = append(out, '\t')
		}
		if rewriteAlgorithms {
			out = append(out, prefix...)
			out = append(out, exportedName(function.Name)...)
		} else {
			out = append(out, function.Name...)
		}
		out = append(out, '(')
		for j := 0; j < len(function.Parameters); j++ {
			if j != 0 {
				out = append(out, ", "...)
			}
			out = append(out, function.Parameters[j]...)
		}
		out = append(out, ")\n}\n"...)
	}
	return out
}

func declarationBlock(declaration Declaration, name string) (Statement, bool) {
	for i := 0; i < len(declaration.Statements); i++ {
		if statementBlockName(declaration.Statements[i]) == name {
			return declaration.Statements[i], true
		}
	}
	return Statement{}, false
}

func findEmbeddedFunction(document Document, name string) (embeddedFunction, bool) {
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		if declaration.Kind != DeclGo {
			continue
		}
		wrapped := make([]byte, 0, len(declaration.GoSource)+16)
		wrapped = append(wrapped, "package backend\n"...)
		wrapped = append(wrapped, declaration.GoSource...)
		file := syntax.ParseFile(wrapped)
		if !file.Ok {
			continue
		}
		for j := 0; j < len(file.Funcs); j++ {
			fn := file.Funcs[j]
			fnName := string(syntax.TokenText(wrapped, file.Tokens[fn.NameTok]))
			if fnName != name {
				continue
			}
			start := file.Tokens[fn.ParamsStart].Start
			end := file.Tokens[fn.ResultEnd].Start
			signature := make([]byte, end-start)
			copy(signature, wrapped[start:end])
			for len(signature) > 0 && (signature[len(signature)-1] == ' ' ||
				signature[len(signature)-1] == '\t' || signature[len(signature)-1] == '\n' ||
				signature[len(signature)-1] == '\r') {
				signature = signature[:len(signature)-1]
			}
			return embeddedFunction{
				Name:       name,
				Signature:  signature,
				Parameters: functionParameterNames(file, fn),
				HasResult:  fn.ResultStart != fn.ResultEnd,
			}, true
		}
	}
	return embeddedFunction{}, false
}

func functionParameterNames(file syntax.File, fn syntax.FuncDecl) []string {
	var names []string
	start := fn.ParamsStart + 1
	depth := 0
	for i := start; i < fn.ParamsEnd-1; i++ {
		text := string(syntax.TokenText(file.Src, file.Tokens[i]))
		if text == "(" || text == "[" {
			depth++
		} else if text == ")" || text == "]" {
			depth--
		}
		if text == "," && depth == 0 {
			if start < i {
				names = append(names, string(syntax.TokenText(file.Src, file.Tokens[start])))
			}
			start = i + 1
		}
	}
	if start < fn.ParamsEnd-1 {
		names = append(names, string(syntax.TokenText(file.Src, file.Tokens[start])))
	}
	return names
}

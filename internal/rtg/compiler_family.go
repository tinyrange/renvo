package rtg

// compiler_family declares the lowering contract of a machine or image, not
// its public spelling. Existing direct-emitter definitions default to native_v1.
func decodeCompilerFamily(declaration Declaration) (string, bool) {
	family := BackendFamilyNativeV1
	seen := false
	for _, statement := range declaration.Statements {
		if len(statement.Tokens) == 0 || statement.Tokens[0] != "compiler_family" {
			continue
		}
		left, right, assignment := statementAssignment(statement)
		if seen || !assignment || len(left) != 1 || len(right) != 1 || len(statement.Children) != 0 {
			return "", false
		}
		family = valueName(right[0])
		if family != BackendFamilyNativeV1 && family != BackendFamilyStructured32 {
			return "", false
		}
		seen = true
	}
	return family, true
}

func validateCompilerFamily(document Document, declaration Declaration) []Diagnostic {
	if _, ok := decodeCompilerFamily(declaration); !ok {
		return []Diagnostic{resolveDiagnostic(document, declaration, "RTG-VALIDATE-135",
			"compiler_family must be one native_v1 or structured32 assignment")}
	}
	return nil
}

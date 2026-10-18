package link

import (
	"renvo.dev/internal/check"
	"renvo.dev/internal/syntax"
	"renvo.dev/internal/unit"
)

// Resolve a field at its shallowest embedding depth. Count paths (capped at
// two) so diamond embeddings remain ambiguous, while visited levels terminate
// recursive pointer embeddings. Alias expansion never changes an emitted type.
func ordinaryPromotedFieldType(program *unit.Program, owner, name string) string {
	current := []string{functionValueBareType(owner)}
	paths := []int{1}
	seen := map[string]bool{}
	for len(current) > 0 {
		var next []string
		var nextPaths []int
		matches := 0
		result := ""
		for i, typ := range current {
			if seen[typ] {
				continue
			}
			if functionValueTypeHasDirectMethod(program, typ, name) {
				matches += paths[i]
				result = "" // Methods win over fields at greater embedding depths.
			}
			underlying := ordinaryUnderlyingType(program, typ, 0)
			if !functionValueHasPrefix(underlying, "struct") {
				continue
			}
			file := syntax.ParseFile([]byte("package main\ntype Owner " + underlying + "\n"))
			if !file.Ok || len(file.Decls) != 1 {
				continue
			}
			start := file.Decls[0].NameTok + 3 // Owner struct {
			fields := check.StructFields(&file, start, file.Decls[0].EndTok-1)
			for _, field := range fields {
				if field.TypeStart < 0 || field.TypeEnd <= field.TypeStart {
					continue
				}
				first, last := file.Tokens[field.TypeStart], file.Tokens[field.TypeEnd-1]
				fieldType := string(file.Src[int(first.Start):int(last.End)])
				fieldName := field.Name
				if field.NameTok < 0 {
					fieldName = functionValueBareType(fieldType)
				}
				if fieldName == name {
					matches += paths[i]
					result = fieldType
				}
				if field.NameTok >= 0 {
					continue
				}
				embedded := functionValueBareType(fieldType)
				if seen[embedded] {
					continue
				}
				slot := -1
				for j, t := range next {
					if t == embedded {
						slot = j
						break
					}
				}
				if slot < 0 {
					next = append(next, embedded)
					nextPaths = append(nextPaths, paths[i])
				} else {
					nextPaths[slot] += paths[i]
					if nextPaths[slot] > 2 {
						nextPaths[slot] = 2
					}
				}
			}
		}
		if matches == 1 {
			return result
		}
		if matches > 1 {
			return ""
		}
		for _, typ := range current {
			seen[typ] = true
		}
		current, paths = next, nextPaths
	}
	return ""
}

package check

import "renvo.dev/internal/syntax"

type genericArrayLengthSource struct {
	pkg    int
	file   int
	change genericReplacement
}

// Alias consumers need the checked length rather than a constant's spelling.
// Keep expressions that syntactically use a local variable so folding cannot
// introduce an unused-variable error in the concrete source.
func (e *genericEnvironment) recordArrayLength(scope genericTypeScope, start int, end int, value wideConstant, bindings []scopedTypeBinding) {
	file := &e.graph.Packages[scope.pkg].Files[scope.file].File
	if start+1 == end && file.Tokens[start].KindLine&255 == syntax.TokenNumber && wideIntegerLiteral(tokenString(file, start)).ok {
		return
	}
	specializer := genericSpecializer{environment: e}
	context := genericExpressionContext{scope: scope, specializer: &specializer, bindings: bindings}
	if context.constantUsesVariable(start, end) {
		return
	}
	change := genericReplacement{start: int(file.Tokens[start].Start), end: int(file.Tokens[end-1].End), text: genericIntegerHex(value)}
	e.recordArrayLengthChange(scope, change)
}

func (e *genericEnvironment) recordArrayLengthChange(scope genericTypeScope, change genericReplacement) {
	for _, source := range e.arrayLengths {
		if source.pkg == scope.pkg && source.file == scope.file && source.change.start == change.start {
			return
		}
	}
	e.arrayLengths = append(e.arrayLengths, genericArrayLengthSource{pkg: scope.pkg, file: scope.file, change: change})
}

func (e *genericEnvironment) appendArrayLengthChanges(pkg int, file int, start int, end int, changes []genericReplacement) []genericReplacement {
	for _, source := range e.arrayLengths {
		if source.pkg == pkg && source.file == file && source.change.start >= start && source.change.end <= end {
			changes = append(changes, source.change)
		}
	}
	return changes
}

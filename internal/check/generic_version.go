package check

import "renvo.dev/internal/load"

func (e *genericEnvironment) versionBefore(scope genericTypeScope, minimum string) bool {
	return load.GoVersionBefore(e.graph.Packages[scope.pkg].Files[scope.file].GoVersion, minimum)
}

func (e *genericEnvironment) requireVersion(scope genericTypeScope, token int, minimum string, feature string) bool {
	if e.versionBefore(scope, minimum) {
		e.fail(scope, token, feature+" requires Go "+minimum+" or later")
		return false
	}
	return true
}

// Constraint satisfaction at an instantiation belongs to its source file's
// language version, which can differ from the generic declaration's version.
func (e *genericEnvironment) argumentSatisfies(scope genericTypeScope, id int, constraint genericConstraint) bool {
	if !e.satisfies(id, constraint) {
		return false
	}
	if constraint.comparable && e.types.get(id).kind != genericParameter && e.versionBefore(scope, "1.20") {
		return e.types.comparable(id, true)
	}
	return true
}

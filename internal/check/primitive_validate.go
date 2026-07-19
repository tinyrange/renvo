package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

func invalidDefiniteLiteralBinary(file syntax.File, op int, left string, right string) bool {
	if tokenTextIs(&file, op, "&&") || tokenTextIs(&file, op, "||") {
		return left != "bool" || right != "bool"
	}
	if tokenTextIs(&file, op, "==") || tokenTextIs(&file, op, "!=") || tokenTextIs(&file, op, "<") || tokenTextIs(&file, op, "<=") || tokenTextIs(&file, op, ">") || tokenTextIs(&file, op, ">=") {
		return left != right
	}
	if tokenTextIs(&file, op, "+") && left == "string" && right == "string" {
		return false
	}
	return left != "int" || right != "int"
}

func invalidDefinitePrimitiveCallAt(pkg *load.Package, info *PackageInfo, fileIndex int, callee int, close int) int {
	file := &pkg.Files[fileIndex].File
	targetFile, target, ok := findDefinitePackageFunc(pkg, info, file, callee)
	if !ok {
		return -1
	}
	params := buildFuncSignature(pkg.Files[targetFile].File, target).Params
	start := callee + 2
	for param := 0; param < len(params) && start < close-1; param++ {
		end := nextDefiniteCallComma(file, start, close-1)
		if end-start == 1 {
			want := definiteBuiltinTypeSpanName(pkg, info, targetFile, params[param].TypeStart, params[param].TypeEnd, 0)
			if primitiveTypeMismatch(want, definiteLiteralKind(*file, start)) {
				return start
			}
		}
		start = end + 1
	}
	return -1
}

func primitiveTypeMismatch(want string, got string) bool {
	if want == "" || got == "" {
		return false
	}
	if want == "string" || want == "bool" {
		return want != got
	}
	if want == "int" || want == "int8" || want == "int16" || want == "int32" || want == "int64" {
		return got != "int"
	}
	if want == "uint" || want == "uint8" || want == "uint16" || want == "uint32" || want == "uint64" || want == "uintptr" {
		return got != "int"
	}
	return false
}

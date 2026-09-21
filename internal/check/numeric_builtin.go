package check

import (
	"renvo.dev/internal/load"
)

func invalidNumericBuiltinCall(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, bindings []scopedTypeBinding, name string, callee, close int, args []ExprSpan) (int, int) {
	file := pkg.Files[fileIndex].File
	count := 1
	if name == "complex" {
		count = 2
	}
	if len(args) != count || tokenTextIs(&file, close-2, "...") {
		return CheckErrBuiltinArity, callee
	}
	identity := ""
	for _, arg := range args {
		value := numericBuiltinExprValue(pkg, info, fileIndex, scope, bindings, arg.StartTok, arg.EndTok, callee, 0)
		if value.kind == "string" || value.kind == "bool" || value.kind == "other" {
			return CheckErrBuiltinOperand, arg.StartTok
		}
		if !value.typed || value.kind == "" {
			continue
		}
		if name != "complex" {
			if value.kind != "complex" {
				return CheckErrBuiltinOperand, arg.StartTok
			}
		} else {
			if value.kind != "float" || identity != "" && value.identity != "" && identity != value.identity {
				return CheckErrBuiltinOperand, arg.StartTok
			}
			identity = value.identity
		}
	}
	return CheckOK, -1
}

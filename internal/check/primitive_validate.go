package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

const (
	definitePrimitiveUnknown = iota
	definitePrimitiveInt
	definitePrimitiveString
	definitePrimitiveBool
)

// Three bits per entry keeps ten cached parameter kinds portable even when the
// frontend executable itself uses 32-bit ints. Later parameters remain unknown
// to this deliberately conservative fast-path and continue to backend checking.
const definitePrimitiveParamLimit = 10

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

func invalidDefinitePrimitiveCallAt(file *syntax.File, open int, close int, target *definiteCallTarget) int {
	start := open + 1
	for param := 0; param < target.primitiveParamCount && start < close-1; param++ {
		end := nextDefiniteCallComma(file, start, close-1)
		if end-start == 1 {
			want := target.primitiveParamCodes >> (param * 3) & 7
			if primitiveCodeMismatch(want, definiteLiteralKind(*file, start)) {
				return start
			}
		}
		start = end + 1
	}
	return -1
}

func prepareDefinitePrimitiveCallTarget(pkg *load.Package, info *PackageInfo, symbolIndex int, target *definiteCallTarget) {
	if target.primitiveReady {
		return
	}
	target.primitiveReady = true
	if symbolIndex < 0 || symbolIndex >= len(info.Symbols) {
		return
	}
	symbol := info.Symbols[symbolIndex]
	if symbol.Kind != SymbolFunc || symbol.File < 0 || symbol.File >= len(pkg.Files) {
		return
	}
	file := pkg.Files[symbol.File].File
	fn, ok := findDefinitePackageFuncDecl(file, symbol.Token)
	if !ok {
		return
	}
	for param := 0; param < definitePrimitiveParamLimit; param++ {
		start, end, found := definitePrimitiveParamSpan(file, fn, param)
		if !found {
			break
		}
		name := definiteBuiltinTypeSpanName(pkg, info, symbol.File, start, end, 0)
		target.primitiveParamCodes = target.primitiveParamCodes | definitePrimitiveTypeCode(name)<<(param*3)
		target.primitiveParamCount++
	}
}

// definitePrimitiveParamSpan resolves one parameter directly from the token
// stream. Unlike buildFuncSignature it does not allocate a complete field list
// for every call site, which is important while the compiler checks itself.
func definitePrimitiveParamSpan(file syntax.File, fn syntax.FuncDecl, wanted int) (int, int, bool) {
	start := fn.ParamsStart + 1
	end := fn.ParamsEnd - 1
	pendingStart := start
	pending := 0
	for start < end {
		segmentEnd := nextTopLevelComma(file, start, end)
		first, last := trimFieldSpan(file, start, segmentEnd)
		if first >= last {
			start = segmentEnd + 1
			continue
		}
		if isSingleIdent(file, first, last) {
			if pending == 0 {
				pendingStart = first
			}
			pending++
			start = segmentEnd + 1
			continue
		}
		if file.Tokens[first].KindLine&255 == syntax.TokenIdent && first+1 < last && !tokCharIs(&file, first+1, '.') {
			if wanted < pending+1 {
				return first + 1, last, true
			}
			wanted -= pending + 1
			pending = 0
			start = segmentEnd + 1
			continue
		}
		if wanted < pending {
			return definitePendingParamSpan(file, pendingStart, first, wanted)
		}
		wanted -= pending
		pending = 0
		if wanted == 0 {
			return first, last, true
		}
		wanted--
		start = segmentEnd + 1
	}
	if wanted < pending {
		return definitePendingParamSpan(file, pendingStart, end, wanted)
	}
	return -1, -1, false
}

func definitePendingParamSpan(file syntax.File, start int, end int, wanted int) (int, int, bool) {
	for i := start; i < end; i++ {
		if file.Tokens[i].KindLine&255 == syntax.TokenIdent {
			if wanted == 0 {
				return i, i + 1, true
			}
			wanted--
		}
	}
	return -1, -1, false
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

func definitePrimitiveTypeCode(name string) int {
	if name == "string" {
		return definitePrimitiveString
	}
	if name == "bool" {
		return definitePrimitiveBool
	}
	if name == "int" || name == "int8" || name == "int16" || name == "int32" || name == "int64" || name == "uint" || name == "uint8" || name == "uint16" || name == "uint32" || name == "uint64" || name == "uintptr" {
		return definitePrimitiveInt
	}
	return definitePrimitiveUnknown
}

func primitiveCodeMismatch(want int, got string) bool {
	if want == definitePrimitiveString {
		return got != "" && got != "string"
	}
	if want == definitePrimitiveBool {
		return got != "" && got != "bool"
	}
	if want == definitePrimitiveInt {
		return got != "" && got != "int"
	}
	return false
}

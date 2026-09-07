package check

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

// Scalar assignment requires identical types for typed operands, whereas
// untyped constants may acquire the destination type. Unknown type identities
// and general constant representability still require broader type checking.
func invalidScalarAppendValue(pkg load.Package, info PackageInfo, fileIndex int, scope CoreScope, bindings []scopedTypeBinding, before int, destination string, source numericBuiltinValue, file syntax.File, arg ExprSpan) bool {
	underlying := destination
	if len(destination) > 6 && destination[:6] == "named:" {
		index := LookupType(info, destination[6:])
		if index < 0 {
			return false
		}
		typ := info.Types[index]
		underlying = conversionUnderlyingType(pkg, info, typ.File, CoreScope{}, typ.TypeStart, typ.TypeEnd, 0)
	}
	want := scalarAppendKind(underlying)
	if want == "int" {
		context := constantIndexContext{pkg: &pkg, info: &info, fileIndex: fileIndex, strict: true, bindings: bindings, before: before}
		value := arrayLiteralConstant(context, arg.StartTok, arg.EndTok, scope)
		if integerConstantOutsideType(value, underlying) {
			return true
		}
	}
	if want == "" || source.kind == "" {
		return false
	}
	if source.typed {
		identity := source.identity
		if identity == "" && (source.kind == "string" || source.kind == "bool") {
			identity = source.kind // default type of an inferred variable
		}
		if identity == "byte" {
			identity = "uint8"
		}
		if identity == "rune" {
			identity = "int32"
		}
		return identity != "" && identity != destination
	}
	if source.kind == "other" {
		return true // untyped nil cannot initialize a scalar element
	}
	if want == "string" || want == "bool" || source.kind == "string" || source.kind == "bool" {
		return want != source.kind
	}
	return want == "int" && unsafeAddFractionalDecimal(file, arg.StartTok, arg.EndTok)
}

// Fixed-width integer bounds are target independent. For machine-sized types,
// reject values outside every supported target's range (at most 64 bits);
// narrower target-specific checks remain part of target-aware typing.
func integerConstantOutsideType(value wideConstant, name string) bool {
	if !value.ok {
		return false
	}
	bits := 64
	if name == "int8" || name == "uint8" || name == "byte" {
		bits = 8
	}
	if name == "int16" || name == "uint16" {
		bits = 16
	}
	if name == "int32" || name == "uint32" || name == "rune" {
		bits = 32
	}
	unsigned := name == "byte" || len(name) >= 4 && name[:4] == "uint"
	if unsigned && value.negative {
		return true
	}
	if !unsigned {
		bits--
	}
	limit := wideShift(wideSmall(1), bits, true)
	comparison := wideMagnitudeCompare(value, limit)
	if value.negative {
		return comparison > 0
	}
	return comparison >= 0
}

func scalarAppendKind(name string) string {
	if name == "string" || name == "bool" {
		return name
	}
	if name == "float32" || name == "float64" {
		return "float"
	}
	if name == "complex64" || name == "complex128" {
		return "complex"
	}
	if definitePrimitiveTypeCode(name) == definitePrimitiveInt || name == "byte" || name == "rune" {
		return "int"
	}
	return ""
}

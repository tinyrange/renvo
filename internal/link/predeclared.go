package link

// A package declaration shadows the universe scope. Its checked references
// must no longer have a spelling recognized as an intrinsic by later passes.
func corePredeclaredAliasNeeded(name string) bool {
	switch name {
	case "append", "cap", "clear", "close", "complex", "copy", "delete", "imag", "len", "make", "max", "min", "new", "panic", "print", "println", "real", "recover",
		"any", "bool", "byte", "complex64", "complex128", "error", "float32", "float64", "int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "true", "false", "nil", "iota":
		return true
	}
	return false
}

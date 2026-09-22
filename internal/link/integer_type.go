package link

func mapLowerIntegerKey(key string) bool {
	return key == "int" || key == "uint" || key == "uintptr" || key == "byte" || key == "rune" ||
		key == "int8" || key == "int16" || key == "int32" || key == "int64" ||
		key == "uint8" || key == "uint16" || key == "uint32" || key == "uint64"
}

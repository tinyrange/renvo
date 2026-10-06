package load

// Format generated source indices without importing the complete strconv
// package (including floating-point conversion) into the self-hosted loader.
func decimalAssemblyIndex(index int) string {
	var digits [20]byte
	at := len(digits)
	value := uint(index)
	if index < 0 {
		value = 0 - value
	}
	for {
		at--
		digits[at] = byte('0' + value%10)
		value /= 10
		if value == 0 {
			break
		}
	}
	if index < 0 {
		at--
		digits[at] = '-'
	}
	return string(digits[at:])
}
func cAssemblyNamespace(path string) string {
	// Full path bytes encode injectively into a safe identifier. Sanitizing only
	// punctuation would collide for names such as a-b.c and a_b.c.
	const digits = "0123456789abcdef"
	result := []byte("cfile_")
	for _, c := range []byte(path) {
		result = append(result, digits[c>>4], digits[c&15])
	}
	return string(result)
}

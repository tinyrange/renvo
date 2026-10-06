package load

import "strconv"

func decimalAssemblyIndex(index int) string { return strconv.Itoa(index) }
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

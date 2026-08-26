package unit

// HasRTGAssembly reports whether a canonical unit carries project assembly.
// Decoding and materializing that table is host-only CompilerJIT work.
func HasRTGAssembly(data []byte) bool {
	if !validUnitRoot(data) {
		return false
	}
	for at := 14; at+6 <= len(data); {
		tag := int(data[at]) | int(data[at+1])<<8
		size := int(data[at+2]) | int(data[at+3])<<8 | int(data[at+4])<<16 | int(data[at+5])<<24
		at += 6
		if size < 0 || at+size < at || at+size > len(data) {
			return false
		}
		if tag == TagRTGAssembly {
			return true
		}
		at += size
	}
	return false
}

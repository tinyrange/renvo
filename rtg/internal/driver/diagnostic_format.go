package driver

// FormatDiagnostic produces the common CLI representation without relying on
// fmt, so the host and self-hosted frontend use exactly the same text.
func FormatDiagnostic(d Diagnostic) string {
	if !d.Valid() {
		return "rtg: error RTG-UNKNOWN-001 (unknown): compilation failed\n"
	}
	out := ""
	if d.Path != "" {
		out = d.Path
		if d.Line > 0 {
			out = out + ":" + diagnosticIntText(d.Line)
			if d.Column > 0 {
				out = out + ":" + diagnosticIntText(d.Column)
			}
		}
		out = out + ": "
	} else {
		out = "rtg: "
	}
	out = out + "error " + d.Code
	if d.Phase != "" {
		out = out + " (" + d.Phase + ")"
	}
	out = out + ": " + d.Message + "\n"
	return out
}

func diagnosticIntText(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var digits []byte
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	out := ""
	if negative {
		out = "-"
	}
	for i := len(digits) - 1; i >= 0; i-- {
		out = out + string(digits[i:i+1])
	}
	return out
}

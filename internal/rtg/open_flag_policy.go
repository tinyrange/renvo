package rtg

// Open flag encodings are a bounded runtime protocol, not an OS-name lookup.
// Omission keeps the portable runtime's 0/1/2/64/512 convention.
func decodeOpenFlagLayout(runtime Declaration) (bool, bool) {
	seen := false
	bsd := false
	for _, statement := range runtime.Statements {
		if len(statement.Tokens) == 0 || statement.Tokens[0] != "open_flag_layout" {
			continue
		}
		left, right, assignment := statementAssignment(statement)
		if seen || !assignment || len(left) != 1 || len(right) != 1 || len(statement.Children) != 0 {
			return false, false
		}
		layout := valueName(right[0])
		if layout != "portable" && layout != "bsd" {
			return false, false
		}
		seen = true
		bsd = layout == "bsd"
	}
	return bsd, true
}

func appendPreparedOpenFlags(source []byte, runtime Declaration) []byte {
	bsd, _ := decodeOpenFlagLayout(runtime) // validated before generation
	create := 64
	truncate := 512
	if bsd {
		create = 512
		truncate = 1024
	}
	source = append(source, "\nconst renvoRTGOpenCreate = "...)
	source = appendDecimalFrame(source, create)
	source = append(source, "\nconst renvoRTGOpenTruncate = "...)
	source = appendDecimalFrame(source, truncate)
	return append(source, '\n')
}

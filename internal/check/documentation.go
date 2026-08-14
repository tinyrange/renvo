package check

// sourceDocumentation returns the contiguous line-comment block immediately
// above the declaration containing offset. Syntax tokens intentionally discard
// comments, so editor queries recover documentation from the original bytes.
func sourceDocumentation(source []byte, offset int) string {
	if offset < 0 || offset > len(source) {
		return ""
	}
	lineStart := offset
	for lineStart > 0 && source[lineStart-1] != '\n' {
		lineStart--
	}
	var lines []string
	cursor := lineStart
	for cursor > 0 {
		end := cursor - 1
		if end > 0 && source[end-1] == '\r' {
			end--
		}
		start := end
		for start > 0 && source[start-1] != '\n' {
			start--
		}
		left := start
		for left < end && (source[left] == ' ' || source[left] == '\t') {
			left++
		}
		if left+2 > end || source[left] != '/' || source[left+1] != '/' {
			break
		}
		left += 2
		if left < end && source[left] == ' ' {
			left++
		}
		lines = append(lines, string(source[left:end]))
		cursor = start
	}
	text := ""
	for i := len(lines) - 1; i >= 0; i-- {
		if text != "" {
			text += "\n"
		}
		text += lines[i]
	}
	return text
}

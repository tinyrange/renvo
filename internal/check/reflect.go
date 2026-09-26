package check

import "renvo.dev/internal/syntax"

// StructFields parses checked struct fields, including unnamed embedded fields.
// Metadata consumers read tags separately, keeping ordinary signatures compact.
func StructFields(file *syntax.File, start int, end int) []Field {
	return parseStructFields(*file, start, end)
}

// StructFieldTag reads a field tag directly from its source token span.
func StructFieldTag(file *syntax.File, field Field) string {
	if field.TypeEnd < 0 || field.TypeEnd >= len(file.Tokens) || file.Tokens[field.TypeEnd].KindLine&255 != syntax.TokenString {
		return ""
	}
	tag, _ := syntax.StringLiteralValue(file.Src, file.Tokens[field.TypeEnd])
	return tag
}

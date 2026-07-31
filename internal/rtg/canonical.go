package rtg

type canonicalDeclaration struct {
	kind string
	name string
	pkg  string
	body []byte
}

// Canonical returns a stable semantic byte stream for a parsed definition.
// Whitespace, comments, source paths, top-level declaration order, and the
// placement of go backend blocks do not contribute to the stream. Declaration
// bodies remain ordered because table order is allowed to be meaningful.
func Canonical(document Document) []byte {
	var out []byte
	out = appendFramed(out, "rtg")
	out = appendDecimalFrame(out, document.Version)
	out = appendFramed(out, document.Unit)

	implements := make([]string, len(document.Implements))
	copy(implements, document.Implements)
	sortStrings(implements)
	for i := 0; i < len(implements); i++ {
		out = appendFramed(out, "implements")
		out = appendFramed(out, implements[i])
	}

	declarations := make([]canonicalDeclaration, 0, len(document.Declarations))
	hasPackages := hasSemanticVirtualPackages(document)
	for i := 0; i < len(document.Declarations); i++ {
		declaration := document.Declarations[i]
		body := canonicalRange(document, declaration.BodyStart, declaration.BodyEnd)
		declarations = append(declarations, canonicalDeclaration{
			kind: declaration.Kind,
			name: declaration.Name,
			pkg:  semanticVirtualPackage(document, declaration),
			body: body,
		})
	}
	for i := 1; i < len(declarations); i++ {
		value := declarations[i]
		at := i
		for at > 0 && canonicalDeclarationLess(value, declarations[at-1]) {
			declarations[at] = declarations[at-1]
			at--
		}
		declarations[at] = value
	}
	for i := 0; i < len(declarations); i++ {
		out = appendFramed(out, declarations[i].kind)
		out = appendFramed(out, declarations[i].name)
		if hasPackages {
			out = appendFramed(out, declarations[i].pkg)
		}
		out = appendFramedBytes(out, declarations[i].body)
	}
	sequences := canonicalDocumentSequences(document)
	for i := 0; i < len(sequences); i++ {
		out = appendFramed(out, "sequence")
		out = appendFramed(out, sequences[i].arch)
		out = appendFramed(out, sequences[i].name)
		out = appendFramedBytes(out, sequences[i].body)
	}
	return out
}

type canonicalDocumentSequence struct {
	arch string
	name string
	body []byte
}

func canonicalDocumentSequences(document Document) []canonicalDocumentSequence {
	var out []canonicalDocumentSequence
	for declaration := 0; declaration < len(document.Declarations); declaration++ {
		arch := document.Declarations[declaration]
		if arch.Kind != DeclArch {
			continue
		}
		sequences := architectureSequences(arch)
		for i := 0; i < len(sequences); i++ {
			out = append(out, canonicalDocumentSequence{
				arch: arch.Name,
				name: sequences[i].Name,
				body: canonicalSequenceStatement(sequences[i].Statement),
			})
		}
	}
	for i := 1; i < len(out); i++ {
		value := out[i]
		at := i
		for at > 0 && canonicalDocumentSequenceLess(value, out[at-1]) {
			out[at] = out[at-1]
			at--
		}
		out[at] = value
	}
	return out
}

func canonicalDocumentSequenceLess(
	left canonicalDocumentSequence, right canonicalDocumentSequence,
) bool {
	if left.arch != right.arch {
		return stringLess(left.arch, right.arch)
	}
	if left.name != right.name {
		return stringLess(left.name, right.name)
	}
	return bytesLess(left.body, right.body)
}

func canonicalRange(document Document, start int, end int) []byte {
	var out []byte
	for i := 0; i < len(document.Tokens); i++ {
		token := document.Tokens[i]
		if token.Kind == TokenEOF || token.Start < start || token.End > end {
			continue
		}
		text := tokenText(document.Source, token)
		if token.Kind == TokenString {
			if value := unquoteSimple(text); value != "" || len(text) == 2 {
				text = value
			}
		} else if token.Kind == TokenNumber {
			text = normalizeNumber(text)
		}
		out = appendDecimalFrame(out, token.Kind)
		out = appendFramed(out, text)
	}
	return out
}

func normalizeNumber(text string) string {
	end := 0
	for end < len(text) && text[end] >= '0' && text[end] <= '9' {
		end++
	}
	if end == 0 {
		return text
	}
	first := 0
	for first+1 < end && text[first] == '0' {
		first++
	}
	return text[first:end] + text[end:]
}

func canonicalDeclarationLess(left canonicalDeclaration, right canonicalDeclaration) bool {
	if left.kind != right.kind {
		return stringLess(left.kind, right.kind)
	}
	if left.name != right.name {
		return stringLess(left.name, right.name)
	}
	if left.pkg != right.pkg {
		return stringLess(left.pkg, right.pkg)
	}
	return bytesLess(left.body, right.body)
}

func bytesLess(left []byte, right []byte) bool {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for i := 0; i < limit; i++ {
		if left[i] != right[i] {
			return left[i] < right[i]
		}
	}
	return len(left) < len(right)
}

func appendFramed(out []byte, value string) []byte {
	return appendFramedBytes(out, []byte(value))
}

func appendFramedBytes(out []byte, value []byte) []byte {
	out = appendDecimalFrame(out, len(value))
	out = append(out, ':')
	out = append(out, value...)
	return out
}

func appendDecimalFrame(out []byte, value int) []byte {
	if value == 0 {
		return append(out, '0')
	}
	var digits [24]byte
	at := len(digits)
	for value > 0 {
		at--
		digits[at] = byte(value%10) + '0'
		value /= 10
	}
	return append(out, digits[at:]...)
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		value := values[i]
		at := i
		for at > 0 && stringLess(value, values[at-1]) {
			values[at] = values[at-1]
			at--
		}
		values[at] = value
	}
}

func stringLess(left string, right string) bool {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for i := 0; i < limit; i++ {
		if left[i] != right[i] {
			return left[i] < right[i]
		}
	}
	return len(left) < len(right)
}

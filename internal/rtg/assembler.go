package rtg

import (
	"strconv"
	"strings"

	"renvo.dev/internal/asmtext"
)

// ParseTargetAssembler parses a text assembly file into whole-function blocks.
// Opcode spellings and permutations are read from the selected definition.
// Metadata-only text/global/function directives are checked, never emitted.
func ParseTargetAssembler(v TargetVocabulary, syntaxName string, source []byte, filename string) ([]TargetBlock, []Diagnostic) {
	lines, diagnostics := assemblerLines(source, filename)
	if len(diagnostics) != 0 {
		return nil, diagnostics
	}
	if syntaxName == "" && len(v.Syntaxes) != 0 {
		syntaxName = v.Syntaxes[0].Name
	}
	globals := []string{}
	for _, line := range lines {
		head, tail := assemblerHead(line.text)
		if head == ".globl" || head == ".global" {
			for _, name := range strings.Split(tail, ",") {
				name = strings.TrimSpace(name)
				if !frontendIdentifier(name) || stringIndex(globals, name) >= 0 {
					return nil, assemblerDiagnostic(filename, line.span, "invalid or duplicate global function")
				}
				globals = append(globals, name)
			}
		}
	}
	if len(globals) == 0 {
		return nil, assemblerDiagnostic(filename, sourceSpan(source, 0, 0), "assembly requires explicit global function declarations")
	}
	blocks := []TargetBlock{}
	current := ""
	body := []assemblerLine{}
	seen := []string{}
	for _, line := range lines {
		head, tail := assemblerHead(line.text)
		if strings.HasPrefix(head, ".") && !strings.Contains(line.text, ":") {
			switch head {
			case ".intel_syntax", ".att_syntax":
				if current != "" {
					return nil, assemblerDiagnostic(filename, line.span, "syntax switches inside functions are not supported")
				}
				style := "att"
				if head == ".intel_syntax" {
					style = "intel"
					if tail != "noprefix" {
						return nil, assemblerDiagnostic(filename, line.span, "Intel syntax requires noprefix")
					}
				} else if tail != "prefix" && tail != "" {
					return nil, assemblerDiagnostic(filename, line.span, "AT&T syntax requires register prefixes")
				}
				found := false
				for _, syntax := range v.Syntaxes {
					if syntax.Style == style {
						syntaxName = syntax.Name
						found = true
						break
					}
				}
				if !found {
					return nil, assemblerDiagnostic(filename, line.span, "requested syntax is not advertised")
				}
			case ".text":
				if tail != "" {
					return nil, assemblerDiagnostic(filename, line.span, "invalid text directive")
				}
			case ".globl", ".global":
			case ".type":
				parts := strings.Split(tail, ",")
				if len(parts) != 2 || stringIndex(globals, strings.TrimSpace(parts[0])) < 0 || (strings.TrimSpace(parts[1]) != "@function" && strings.TrimSpace(parts[1]) != "%function") {
					return nil, assemblerDiagnostic(filename, line.span, "only declared function symbol types are supported")
				}
			case ".size":
				parts := strings.Split(tail, ",")
				if len(parts) != 2 || strings.TrimSpace(parts[0]) != current || strings.ReplaceAll(strings.TrimSpace(parts[1]), " ", "") != ".-"+current {
					return nil, assemblerDiagnostic(filename, line.span, "unsupported function size expression")
				}
			default:
				return nil, assemblerDiagnostic(filename, line.span, "unsupported assembler directive "+head)
			}
			continue
		}
		colon := strings.IndexByte(line.text, ':')
		if colon >= 0 && stringIndex(globals, strings.TrimSpace(line.text[:colon])) >= 0 {
			if current != "" {
				block, ds := parseTargetInstructionLines(v, syntaxName, body, filename, current)
				if len(ds) != 0 {
					return nil, ds
				}
				blocks = append(blocks, block)
			}
			current = strings.TrimSpace(line.text[:colon])
			if stringIndex(seen, current) >= 0 {
				return nil, assemblerDiagnostic(filename, line.span, "duplicate function definition "+current)
			}
			seen = append(seen, current)
			body = nil
			rest := strings.TrimSpace(line.text[colon+1:])
			if rest != "" {
				body = append(body, assemblerLine{text: rest, span: line.span})
			}
			continue
		}
		if current == "" {
			return nil, assemblerDiagnostic(filename, line.span, "instruction or local label outside a function")
		}
		body = append(body, line)
	}
	if current != "" {
		block, ds := parseTargetInstructionLines(v, syntaxName, body, filename, current)
		if len(ds) != 0 {
			return nil, ds
		}
		blocks = append(blocks, block)
	}
	if len(seen) != len(globals) {
		return nil, assemblerDiagnostic(filename, sourceSpan(source, len(source), len(source)), "declared assembly function has no body")
	}
	return blocks, nil
}

// ParseTargetInstructionBlock is the same spelling frontend without file-level
// directives. Its labels are block-local; runtime operands belong to a separate
// constraint-binding interface rather than being interpolated as expressions.
func ParseTargetInstructionBlock(v TargetVocabulary, syntaxName string, source []byte, filename, name string) (TargetBlock, []Diagnostic) {
	lines, diagnostics := assemblerLines(source, filename)
	if len(diagnostics) != 0 {
		return TargetBlock{}, diagnostics
	}
	return parseTargetInstructionLines(v, syntaxName, lines, filename, name)
}

type assemblerLine struct {
	text string
	span Span
}
type assemblerLabel struct {
	name, handle string
	line         int
	numeric      bool
}

func assemblerDiagnostic(filename string, span Span, message string) []Diagnostic {
	return []Diagnostic{{Filename: filename, Span: span, Code: "RTG-ASSEMBLER-001", Message: message}}
}
func assemblerHead(text string) (string, string) {
	at := strings.IndexAny(text, " \t\r")
	if at < 0 {
		return text, ""
	}
	return text[:at], strings.TrimSpace(text[at:])
}
func assemblerLines(source []byte, filename string) ([]assemblerLine, []Diagnostic) {
	if at := invalidUTF8Offset(source); at >= 0 {
		return nil, assemblerDiagnostic(filename, sourceSpan(source, at, at+1), "assembly is not valid UTF-8")
	}
	scanned, err := asmtext.Scan(source)
	if err.Message != "" {
		return nil, assemblerDiagnostic(filename, sourceSpan(source, err.Offset, err.Offset), err.Message)
	}
	lines := make([]assemblerLine, 0, len(scanned))
	cursor := Position{Line: 1, Column: 1}
	for _, line := range scanned {
		start := advancePosition(source, cursor, line.Start)
		cursor = advancePosition(source, start, line.End)
		lines = append(lines, assemblerLine{text: line.Text, span: Span{Start: start, End: cursor}})
	}
	return lines, nil
}

func assemblerSymbol(name string) bool {
	if name == "" {
		return false
	}
	for i, c := range []byte(name) {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c == '.' || c >= '0' && c <= '9' && i > 0) {
			return false
		}
	}
	return true
}
func assemblerNumeric(name string) bool {
	if len(name) == 0 {
		return false
	}
	for _, c := range []byte(name) {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
func splitAssemblerOperands(text string) ([]string, bool) {
	if text == "" {
		return nil, true
	}
	out := []string{}
	start, paren, bracket := 0, 0, 0
	for at, c := range []byte(text) {
		switch c {
		case '(':
			paren++
		case ')':
			paren--
		case '[':
			bracket++
		case ']':
			bracket--
		case ',':
			if paren == 0 && bracket == 0 {
				word := strings.TrimSpace(text[start:at])
				if word == "" {
					return nil, false
				}
				out = append(out, word)
				start = at + 1
			}
		}
		if paren < 0 || bracket < 0 || paren > 1 || bracket > 1 {
			return nil, false
		}
	}
	word := strings.TrimSpace(text[start:])
	if word == "" || paren != 0 || bracket != 0 {
		return nil, false
	}
	return append(out, word), true
}
func parseTargetInstructionLines(v TargetVocabulary, syntaxName string, lines []assemblerLine, filename, name string) (TargetBlock, []Diagnostic) {
	block := TargetBlock{Name: name}
	if !v.Ok {
		if len(v.Diagnostics) != 0 {
			return block, v.Diagnostics
		}
		return block, assemblerDiagnostic(filename, Span{}, "target vocabulary unavailable")
	}
	var syntax TargetSyntax
	found := false
	for _, s := range v.Syntaxes {
		if s.Name == syntaxName {
			syntax, found = s, true
		}
	}
	if !found {
		return block, assemblerDiagnostic(filename, Span{}, "assembler syntax is not advertised by the selected backend: "+syntaxName)
	}
	labels := []assemblerLabel{}
	instructions := []assemblerLine{}
	for _, line := range lines {
		text := line.text
		if colon := strings.IndexByte(text, ':'); colon >= 0 {
			symbol := strings.TrimSpace(text[:colon])
			numeric := assemblerNumeric(symbol)
			if !numeric && !assemblerSymbol(symbol) {
				return block, assemblerDiagnostic(filename, line.span, "invalid assembler label")
			}
			for _, old := range labels {
				if !numeric && old.name == symbol {
					return block, assemblerDiagnostic(filename, line.span, "duplicate assembler label "+symbol)
				}
			}
			labels = append(labels, assemblerLabel{name: symbol, handle: "asmLabel" + strconv.Itoa(len(labels)), line: len(instructions), numeric: numeric})
			instructions = append(instructions, assemblerLine{text: ":" + labels[len(labels)-1].handle, span: line.span})
			text = strings.TrimSpace(text[colon+1:])
		}
		if text != "" {
			instructions = append(instructions, assemblerLine{text: text, span: line.span})
		}
	}
	if len(labels) != 0 {
		if syntax.NewLabel == "" || syntax.BindLabel == "" {
			return block, assemblerDiagnostic(filename, Span{}, "selected syntax does not declare label operations")
		}
		for _, label := range labels {
			block.Instructions = append(block.Instructions, TargetInstruction{Operation: syntax.NewLabel, Result: label.handle})
		}
	}
	for index, line := range instructions {
		if strings.HasPrefix(line.text, ":") {
			block.Instructions = append(block.Instructions, TargetInstruction{Operation: syntax.BindLabel, Operands: []TargetOperand{{Kind: "value", Value: line.text[1:]}}, Span: line.span})
			continue
		}
		mnemonic, tail := assemblerHead(line.text)
		operands, ok := splitAssemblerOperands(tail)
		if !ok {
			return block, assemblerDiagnostic(filename, line.span, "invalid assembler operand list")
		}
		matched := false
		var selected TargetSyntaxForm
		var values []TargetOperand
		for _, form := range syntax.Forms {
			if form.Mnemonic != strings.ToLower(mnemonic) || len(form.Parameters) != len(operands) {
				continue
			}
			candidate := []TargetOperand{}
			valid := true
			for i, parameter := range form.Parameters {
				operand, good := assemblerOperand(v, syntax, parameter.Kind, operands[i], labels, index, targetOperationWidth(v, form.Operation))
				if !good {
					valid = false
					break
				}
				candidate = append(candidate, operand)
			}
			if valid {
				if matched {
					return block, assemblerDiagnostic(filename, line.span, "ambiguous assembler operands")
				}
				selected, values, matched = form, candidate, true
			}
		}
		if !matched {
			return block, assemblerDiagnostic(filename, line.span, "unknown or unsupported assembler instruction/operands: "+line.text)
		}
		// Address handles are constructed independently of the consuming opcode.
		// Neither the frontend nor its source has access to encoder implementation.
		for i, operand := range values {
			if operand.Kind == "address" {
				if syntax.Address == "" {
					return block, assemblerDiagnostic(filename, line.span, "address construction is unavailable")
				}
				parts := strings.Split(operand.Value, ",")
				handle := "asmAddress" + strconv.Itoa(len(block.Instructions))
				block.Instructions = append(block.Instructions, TargetInstruction{Operation: syntax.Address, Result: handle, Operands: []TargetOperand{{Kind: "register", Value: parts[0]}, {Kind: "int", Value: parts[1]}}, Span: line.span})
				values[i] = TargetOperand{Kind: "value", Value: handle}
			}
		}
		instruction := TargetInstruction{Operation: selected.Operation, Span: line.span}
		for _, argument := range selected.Arguments {
			if argument.Parameter >= 0 {
				if argument.Parameter >= len(values) {
					return block, assemblerDiagnostic(filename, line.span, "invalid assembler form operand index")
				}
				instruction.Operands = append(instruction.Operands, values[argument.Parameter])
			} else {
				instruction.Operands = append(instruction.Operands, TargetOperand{Kind: argument.LiteralKind, Value: argument.LiteralValue})
			}
		}
		block.Instructions = append(block.Instructions, instruction)
	}
	steps, message := validateTargetBlock(v, block)
	if message != "" {
		span := Span{}
		if len(steps) != 0 {
			span = steps[len(steps)-1].Span
		}
		return block, assemblerDiagnostic(filename, span, message)
	}
	return block, nil
}
func assemblerOperand(v TargetVocabulary, syntax TargetSyntax, kind, text string, labels []assemblerLabel, index, bits int) (TargetOperand, bool) {
	operand := TargetOperand{Kind: kind, Value: text}
	if kind == "label" {
		directional := len(text) > 1 && (text[len(text)-1] == 'f' || text[len(text)-1] == 'b') && assemblerNumeric(text[:len(text)-1])
		best := -1
		for i, label := range labels {
			if !directional && !label.numeric && label.name == text {
				best = i
				break
			}
			if directional && label.numeric && label.name == text[:len(text)-1] {
				forward := text[len(text)-1] == 'f'
				if forward && label.line > index && (best < 0 || label.line < labels[best].line) || !forward && label.line < index && (best < 0 || label.line > labels[best].line) {
					best = i
				}
			}
		}
		if best < 0 {
			return operand, false
		}
		return TargetOperand{Kind: "value", Value: labels[best].handle}, true
	}
	if kind == "address" {
		base, offset, ok := assemblerAddress(syntax.Style, text)
		if !ok {
			return operand, false
		}
		if syntax.Style == "att" {
			if !strings.HasPrefix(base, "%") {
				return operand, false
			}
			base = base[1:]
		}
		canonical, valid := targetLiteral(v, TargetOperand{Kind: "int", Value: offset})
		if !valid || stringIndex(v.Registers, base) < 0 {
			return operand, false
		}
		return TargetOperand{Kind: "address", Value: base + "," + canonical}, true
	}
	if kind == "register" {
		if syntax.Style == "att" {
			if !strings.HasPrefix(text, "%") {
				return operand, false
			}
			text = text[1:]
		}
		register, valid := targetSpellingRegister(v, strings.ToLower(text), bits)
		if !valid {
			return operand, false
		}
		operand.Value = register
	} else if frontendIntegerKind(kind) && syntax.Style == "att" {
		if !strings.HasPrefix(text, "$") {
			return operand, false
		}
		operand.Value = text[1:]
	}
	canonical, ok := targetLiteral(v, operand)
	operand.Value = canonical
	return operand, ok
}
func assemblerAddress(style, text string) (string, string, bool) {
	if style == "intel" {
		if !strings.HasPrefix(text, "[") || !strings.HasSuffix(text, "]") {
			return "", "", false
		}
		text = strings.ReplaceAll(strings.ReplaceAll(text[1:len(text)-1], " ", ""), "\t", "")
		at := strings.IndexAny(text, "+-")
		if at < 0 {
			return strings.ToLower(text), "0", true
		}
		offset := text[at:]
		if offset[0] == '+' {
			offset = offset[1:]
		}
		return strings.ToLower(text[:at]), offset, true
	}
	at := strings.IndexByte(text, '(')
	if at < 0 || !strings.HasSuffix(text, ")") {
		return "", "", false
	}
	offset := strings.TrimSpace(text[:at])
	if offset == "" {
		offset = "0"
	}
	return strings.TrimSpace(text[at+1 : len(text)-1]), offset, true
}

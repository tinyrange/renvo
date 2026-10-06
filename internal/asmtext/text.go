// Package asmtext owns target-independent text and function-symbol discovery.
// Instruction semantics and typed operands belong to the selected frontend.
package asmtext

import "strings"

type Line struct {
	Text       string
	Start, End int
}
type Function struct {
	Name   string
	Offset int
}
type Error struct {
	Offset  int
	Message string
}

func Scan(source []byte) ([]Line, Error) {
	if len(source) > 16*1024*1024 {
		return nil, Error{Message: "assembly exceeds the source limit"}
	}
	lines := []Line{}
	// Keep line storage owned by the scanner. A strings.Builder adds a
	// copy-check panic path to every self-hosted compiler that imports us.
	var text []byte
	start := 0
	blockComment, lineComment := false, false
	for at := 0; at <= len(source); at++ {
		c := byte('\n')
		if at < len(source) {
			c = source[at]
		}
		if blockComment && at+1 < len(source) && c == '*' && source[at+1] == '/' {
			blockComment = false
			text = append(text, ' ')
			at++
			continue
		}
		if c == '\n' || c == ';' && !blockComment && !lineComment {
			value := strings.TrimSpace(string(text))
			if value != "" {
				lines = append(lines, Line{Text: value, Start: start, End: at})
			}
			if len(lines) > 8192 {
				return nil, Error{Offset: start, Message: "assembly exceeds the statement limit"}
			}
			text = nil
			start = at + 1
			lineComment = false
			continue
		}
		if blockComment || lineComment {
			continue
		}
		if c == '#' || at+1 < len(source) && c == '/' && source[at+1] == '/' {
			lineComment = true
			continue
		}
		if at+1 < len(source) && c == '/' && source[at+1] == '*' {
			blockComment = true
			at++
			text = append(text, ' ')
			continue
		}
		if c == '"' || c == '\'' || c == '`' {
			return nil, Error{Offset: at, Message: "quoted data and assembler expressions are not instruction operands"}
		}
		text = append(text, c)
	}
	if blockComment {
		return nil, Error{Offset: start, Message: "unterminated assembler comment"}
	}
	return lines, Error{}
}
func Head(text string) (string, string) {
	at := strings.IndexAny(text, " \t\r")
	if at < 0 {
		return text, ""
	}
	return text[:at], strings.TrimSpace(text[at:])
}
func Identifier(name string) bool {
	if len(name) == 0 {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c >= '0' && c <= '9' && i > 0) {
			return false
		}
	}
	return true
}
func contains(names []string, name string) bool {
	for i := 0; i < len(names); i++ {
		if names[i] == name {
			return true
		}
	}
	return false
}

// Functions returns definition order, not declaration order: unit entry indices
// must agree with the later authoritative parse even for reordered .globl lines.
func Functions(lines []Line) ([]Function, Error) {
	globals := []string{}
	for i := 0; i < len(lines); i++ {
		head, tail := Head(lines[i].Text)
		if head != ".globl" && head != ".global" {
			continue
		}
		names := strings.Split(tail, ",")
		for j := 0; j < len(names); j++ {
			name := strings.TrimSpace(names[j])
			if !Identifier(name) || contains(globals, name) {
				return nil, Error{Offset: lines[i].Start, Message: "invalid or duplicate global function"}
			}
			globals = append(globals, name)
		}
	}
	if len(globals) == 0 {
		return nil, Error{Message: "assembly requires explicit global function declarations"}
	}
	functions := []Function{}
	defined := []string{}
	for i := 0; i < len(lines); i++ {
		text := lines[i].Text
		colon := strings.IndexByte(text, ':')
		if colon < 0 {
			continue
		}
		name := strings.TrimSpace(text[:colon])
		if !contains(globals, name) {
			continue
		}
		if contains(defined, name) {
			return nil, Error{Offset: lines[i].Start, Message: "duplicate function definition"}
		}
		defined = append(defined, name)
		functions = append(functions, Function{Name: name, Offset: lines[i].Start})
	}
	if len(defined) != len(globals) {
		return nil, Error{Message: "declared assembly function has no body"}
	}
	return functions, Error{}
}

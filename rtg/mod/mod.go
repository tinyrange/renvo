package mod

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Module struct {
	Root     string
	Path     string
	Requires []Require
	Replaces []Replace
}

type Require struct {
	Path    string
	Version string
}

type Replace struct {
	Old string
	New string
}

type modError string

func (err modError) Error() string {
	return string(err)
}

func Find(start string) (Module, error) {
	if start == "" {
		start = "."
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		return Module{}, err
	}
	info, err := os.Stat(abs)
	if err == nil && !fileInfoIsDir(info) {
		abs = filepath.Dir(abs)
	}
	for {
		path := filepath.Join(abs, "go.mod")
		if isGoModPath(path) {
			path = "go.mod"
		}
		stablePath := copyString(path)
		parsed, ok, readErr := readModuleFile(stablePath)
		if readErr != nil {
			return Module{}, readErr
		}
		if ok {
			parsedPath := parsed.Path
			parsedRequires := parsed.Requires
			parsedReplaces := parsed.Replaces
			return newModule(abs, parsedPath, parsedRequires, parsedReplaces), nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return Module{}, fmt.Errorf("go.mod not found from %s", start)
		}
		abs = parent
	}
}

func isGoModPath(path string) bool {
	if len(path) != 6 {
		return false
	}
	return path[0] == 'g' && path[1] == 'o' && path[2] == '.' && path[3] == 'm' && path[4] == 'o' && path[5] == 'd'
}

func copyString(s string) string {
	var out []byte
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	return string(out)
}

func newModule(root string, path string, requires []Require, replaces []Replace) Module {
	var module Module
	module.Root = root
	module.Path = path
	module.Requires = requires
	module.Replaces = replaces
	return module
}

func readModuleFile(path string) (Module, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Module{}, false, nil
	}
	text := bytesToString(data)
	parsed, err := ParseFile(text)
	if err != nil {
		return Module{}, false, modError(path + ": " + err.Error())
	}
	return parsed, true, nil
}

func fileInfoIsDir(info os.FileInfo) bool {
	return info.IsDir()
}

func bytesToString(data []byte) string {
	var out []byte
	for i := 0; i < len(data); i++ {
		out = append(out, data[i])
	}
	return string(out)
}

func ParseFile(data string) (Module, error) {
	modulePath := ""
	var requires []Require
	var replaces []Replace
	stripped, err := stripComments(data)
	if err != nil {
		return Module{}, err
	}
	lines := strings.Split(stripped, "\n")
	inRequireBlock := false
	inReplaceBlock := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		fields, err := directiveFields(line)
		if err != nil {
			if inRequireBlock {
				return Module{}, fmt.Errorf("malformed require directive")
			}
			if inReplaceBlock {
				return Module{}, fmt.Errorf("malformed replace directive")
			}
			return Module{}, malformedDirectiveForLine(line)
		}
		if len(fields) == 0 {
			continue
		}
		first := fields[0]
		if inRequireBlock {
			if first == ")" {
				inRequireBlock = false
				continue
			}
			req, err := parseRequireFields(fields)
			if err != nil {
				return Module{}, err
			}
			requires = append(requires, req)
			continue
		}
		if inReplaceBlock {
			if first == ")" {
				inReplaceBlock = false
				continue
			}
			repl, err := parseReplaceFields(fields)
			if err != nil {
				return Module{}, err
			}
			replaces = append(replaces, repl)
			continue
		}
		if first == "module" {
			if modulePath != "" || len(fields) != 2 {
				return Module{}, fmt.Errorf("malformed module directive")
			}
			field := fields[1]
			path, err := unquoteField(field)
			if err != nil {
				return Module{}, fmt.Errorf("malformed module directive")
			}
			modulePath = path
			continue
		}
		if first == "require" {
			if len(fields) == 2 {
				second := fields[1]
				if second == "(" {
					inRequireBlock = true
					continue
				}
			}
			reqFields := fields[1:]
			req, err := parseRequireFields(reqFields)
			if err != nil {
				return Module{}, err
			}
			requires = append(requires, req)
			continue
		}
		if first == "replace" {
			if len(fields) == 2 {
				second := fields[1]
				if second == "(" {
					inReplaceBlock = true
					continue
				}
			}
			replFields := fields[1:]
			repl, err := parseReplaceFields(replFields)
			if err != nil {
				return Module{}, err
			}
			replaces = append(replaces, repl)
		}
	}
	if inRequireBlock {
		return Module{}, fmt.Errorf("malformed require directive")
	}
	if inReplaceBlock {
		return Module{}, fmt.Errorf("malformed replace directive")
	}
	if modulePath == "" {
		return Module{}, fmt.Errorf("module directive not found")
	}
	return newModule("", modulePath, requires, replaces), nil
}

func directiveFields(line string) ([]string, error) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "module ") || trimmed == "module" {
		return strings.Fields(trimmed), nil
	}
	return lineFields(line)
}

func ParseModulePath(data string) (string, error) {
	module, err := ParseFile(data)
	if err != nil {
		return "", err
	}
	return module.Path, nil
}

func parseReplaceFields(fields []string) (Replace, error) {
	arrow := -1
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		if field == "=>" {
			arrow = i
			break
		}
	}
	if arrow <= 0 || arrow+1 >= len(fields) {
		return Replace{}, fmt.Errorf("malformed replace directive")
	}
	oldFields := fields[:arrow]
	newFields := fields[arrow+1:]
	if len(oldFields) > 2 || len(newFields) > 2 {
		return Replace{}, fmt.Errorf("malformed replace directive")
	}
	oldFields, err := unquoteFields(oldFields)
	if err != nil {
		return Replace{}, fmt.Errorf("malformed replace directive")
	}
	newFields, err = unquoteFields(newFields)
	if err != nil {
		return Replace{}, fmt.Errorf("malformed replace directive")
	}
	if invalidReplaceFields(oldFields) || invalidReplaceFields(newFields) {
		return Replace{}, fmt.Errorf("malformed replace directive")
	}
	if len(newFields) == 2 && isLocalReplacePath(newFields[0]) {
		return Replace{}, fmt.Errorf("malformed replace directive")
	}
	return Replace{Old: oldFields[0], New: newFields[0]}, nil
}

func invalidReplaceFields(fields []string) bool {
	if len(fields) == 0 {
		return true
	}
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		if field == "(" || field == ")" || field == "=>" {
			return true
		}
	}
	return false
}

func isLocalReplacePath(path string) bool {
	return filepath.IsAbs(path) || strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") || path == "." || path == ".."
}

func parseRequireFields(fields []string) (Require, error) {
	if len(fields) != 2 {
		return Require{}, fmt.Errorf("malformed require directive")
	}
	var err error
	fields, err = unquoteFields(fields)
	if err != nil {
		return Require{}, fmt.Errorf("malformed require directive")
	}
	if fields[0] == "(" || fields[0] == ")" || fields[1] == "(" || fields[1] == ")" {
		return Require{}, fmt.Errorf("malformed require directive")
	}
	return Require{Path: fields[0], Version: fields[1]}, nil
}

func unquoteFields(fields []string) ([]string, error) {
	out := make([]string, 0, len(fields))
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		unquoted, err := unquoteField(field)
		if err != nil {
			return nil, err
		}
		out = append(out, unquoted)
	}
	return out, nil
}

func unquoteField(field string) (string, error) {
	if len(field) == 0 {
		return field, nil
	}
	if field[0] != '"' && field[0] != '`' {
		return field, nil
	}
	return strconv.Unquote(field)
}

func lineFields(line string) ([]string, error) {
	var fields []string
	for i := 0; i < len(line); {
		for i < len(line) && isFieldSpace(line[i]) {
			i++
		}
		if i >= len(line) {
			break
		}
		start := i
		if line[i] == '"' {
			i++
			for i < len(line) {
				if line[i] == '\\' {
					i += 2
					continue
				}
				if line[i] == '"' {
					i++
					fields = append(fields, line[start:i])
					break
				}
				i++
			}
			if i > len(line) || len(fields) == 0 || fields[len(fields)-1] != line[start:i] {
				return nil, fmt.Errorf("malformed directive")
			}
			continue
		}
		if line[i] == '`' {
			i++
			for i < len(line) && line[i] != '`' {
				i++
			}
			if i >= len(line) {
				return nil, fmt.Errorf("malformed directive")
			}
			i++
			fields = append(fields, line[start:i])
			continue
		}
		for i < len(line) && !isFieldSpace(line[i]) {
			i++
		}
		fields = append(fields, line[start:i])
	}
	return fields, nil
}

func isFieldSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r'
}

func malformedDirectiveForLine(line string) error {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return fmt.Errorf("malformed module directive")
	}
	switch fields[0] {
	case "module":
		return fmt.Errorf("malformed module directive")
	case "require":
		return fmt.Errorf("malformed require directive")
	case "replace":
		return fmt.Errorf("malformed replace directive")
	}
	return fmt.Errorf("malformed module directive")
}

func stripComments(data string) (string, error) {
	var out []byte
	inBlock := false
	for i := 0; i < len(data); i++ {
		if inBlock {
			if i+1 < len(data) && data[i] == '*' && data[i+1] == '/' {
				inBlock = false
				i++
				out = append(out, ' ')
				continue
			}
			if data[i] == '\n' {
				out = append(out, '\n')
			}
			continue
		}
		if i+1 < len(data) && data[i] == '/' && data[i+1] == '/' {
			for i < len(data) && data[i] != '\n' {
				i++
			}
			if i < len(data) {
				out = append(out, data[i])
			}
			continue
		}
		if i+1 < len(data) && data[i] == '/' && data[i+1] == '*' {
			inBlock = true
			i++
			out = append(out, ' ')
			continue
		}
		out = append(out, data[i])
	}
	if inBlock {
		return "", fmt.Errorf("malformed comment")
	}
	return string(out), nil
}

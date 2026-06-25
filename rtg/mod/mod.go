package mod

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Module struct {
	Root     string
	Path     string
	Replaces []Replace
}

type Replace struct {
	Old string
	New string
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
	if err == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}
	for {
		path := filepath.Join(abs, "go.mod")
		data, err := os.ReadFile(path)
		if err == nil {
			parsed, err := ParseFile(string(data))
			if err != nil {
				return Module{}, fmt.Errorf("%s: %w", path, err)
			}
			return Module{Root: abs, Path: parsed.Path, Replaces: parsed.Replaces}, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return Module{}, fmt.Errorf("go.mod not found from %s", start)
		}
		abs = parent
	}
}

func ParseFile(data string) (Module, error) {
	var module Module
	lines := strings.Split(data, "\n")
	inReplaceBlock := false
	for _, line := range lines {
		line = stripLineComment(line)
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if inReplaceBlock {
			if fields[0] == ")" {
				inReplaceBlock = false
				continue
			}
			if repl, ok := parseReplaceFields(fields); ok {
				module.Replaces = append(module.Replaces, repl)
			}
			continue
		}
		if len(fields) >= 2 && fields[0] == "module" {
			module.Path = fields[1]
			continue
		}
		if fields[0] == "replace" {
			if len(fields) >= 2 && fields[1] == "(" {
				inReplaceBlock = true
				continue
			}
			if repl, ok := parseReplaceFields(fields[1:]); ok {
				module.Replaces = append(module.Replaces, repl)
			}
		}
	}
	if module.Path == "" {
		return Module{}, fmt.Errorf("module directive not found")
	}
	return module, nil
}

func ParseModulePath(data string) (string, error) {
	module, err := ParseFile(data)
	if err != nil {
		return "", err
	}
	return module.Path, nil
}

func parseReplaceFields(fields []string) (Replace, bool) {
	arrow := -1
	for i, field := range fields {
		if field == "=>" {
			arrow = i
			break
		}
	}
	if arrow <= 0 || arrow+1 >= len(fields) {
		return Replace{}, false
	}
	return Replace{Old: fields[0], New: fields[arrow+1]}, true
}

func stripLineComment(line string) string {
	for i := 0; i+1 < len(line); i++ {
		if line[i] == '/' && line[i+1] == '/' {
			return line[:i]
		}
	}
	return line
}

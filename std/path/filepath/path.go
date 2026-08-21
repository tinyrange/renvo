package filepath

import (
	"errors"
	"os"
)

const Separator = '/'
const ListSeparator = ':'

var ErrBadPattern = errors.New("syntax error in pattern")
var SkipDir = errors.New("skip this directory")

func IsPathSeparator(c uint8) bool  { return c == Separator }
func VolumeName(path string) string { return "" }
func IsAbs(path string) bool        { return len(path) > 0 && IsPathSeparator(path[0]) }
func ToSlash(path string) string    { return path }
func FromSlash(path string) string  { return path }

func Clean(path string) string {
	if path == "" {
		return "."
	}
	rooted := IsAbs(path)
	parts := splitParts(path)
	stack := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			if len(stack) > 0 && stack[len(stack)-1] != ".." {
				stack = stack[:len(stack)-1]
			} else if !rooted {
				stack = append(stack, part)
			}
		} else {
			stack = append(stack, part)
		}
	}
	out := joinParts(stack)
	if rooted {
		out = "/" + out
	}
	if out == "" {
		if rooted {
			return "/"
		}
		return "."
	}
	return out
}
func Join(elem ...string) string {
	out := ""
	for _, item := range elem {
		if item == "" {
			continue
		}
		if out != "" {
			out += "/"
		}
		out += item
	}
	if out == "" {
		return ""
	}
	return Clean(out)
}
func Split(path string) (dir, file string) {
	for i := len(path) - 1; i >= 0; i-- {
		if IsPathSeparator(path[i]) {
			return path[:i+1], path[i+1:]
		}
	}
	return "", path
}
func Base(path string) string {
	if path == "" {
		return "."
	}
	for len(path) > 1 && IsPathSeparator(path[len(path)-1]) {
		path = path[:len(path)-1]
	}
	_, file := Split(path)
	if file == "" {
		return "/"
	}
	return file
}
func Dir(path string) string { dir, _ := Split(path); return Clean(dir) }
func Ext(path string) string {
	base := Base(path)
	for i := len(base) - 1; i >= 0; i-- {
		if base[i] == '.' {
			return base[i:]
		}
	}
	return ""
}
func Abs(path string) (string, error) {
	if IsAbs(path) {
		return Clean(path), nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return Join(wd, path), nil
}
func Rel(basepath, targpath string) (string, error) {
	base := Clean(basepath)
	target := Clean(targpath)
	if IsAbs(base) != IsAbs(target) {
		return "", errors.New("Rel: can't make target relative to base")
	}
	bp, tp := splitParts(trimRoot(base)), splitParts(trimRoot(target))
	i := 0
	for i < len(bp) && i < len(tp) && bp[i] == tp[i] {
		i++
	}
	out := make([]string, 0, len(bp)-i+len(tp)-i)
	for j := i; j < len(bp); j++ {
		out = append(out, "..")
	}
	out = append(out, tp[i:]...)
	if len(out) == 0 {
		return ".", nil
	}
	return joinParts(out), nil
}

func Match(pattern, name string) (bool, error) { return match(pattern, name) }
func match(pattern, name string) (bool, error) {
	for len(pattern) > 0 {
		if pattern[0] == '*' {
			for len(pattern) > 0 && pattern[0] == '*' {
				pattern = pattern[1:]
			}
			if pattern == "" {
				return !containsSeparator(name), nil
			}
			for i := 0; i <= len(name); i++ {
				if i > 0 && IsPathSeparator(name[i-1]) {
					break
				}
				ok, err := match(pattern, name[i:])
				if err != nil || ok {
					return ok, err
				}
			}
			return false, nil
		}
		if len(name) == 0 {
			return false, nil
		}
		if pattern[0] == '?' {
			if IsPathSeparator(name[0]) {
				return false, nil
			}
			pattern, name = pattern[1:], name[1:]
			continue
		}
		if pattern[0] == '[' {
			end := 1
			for end < len(pattern) && pattern[end] != ']' {
				end++
			}
			if end == len(pattern) {
				return false, ErrBadPattern
			}
			if IsPathSeparator(name[0]) {
				return false, nil
			}
			matched, err := matchClass(pattern[1:end], name[0])
			if err != nil || !matched {
				return false, err
			}
			pattern, name = pattern[end+1:], name[1:]
			continue
		}
		if pattern[0] == 92 {
			if len(pattern) == 1 {
				return false, ErrBadPattern
			}
			pattern = pattern[1:]
		}
		if pattern[0] != name[0] {
			return false, nil
		}
		pattern, name = pattern[1:], name[1:]
	}
	return name == "", nil
}
func matchClass(class string, c byte) (bool, error) {
	negated := false
	if len(class) > 0 && class[0] == '^' {
		negated = true
		class = class[1:]
	}
	if class == "" {
		return false, ErrBadPattern
	}
	matched := false
	for i := 0; i < len(class); i++ {
		if i+2 < len(class) && class[i+1] == '-' {
			if class[i] <= c && c <= class[i+2] {
				matched = true
			}
			i += 2
		} else if class[i] == c {
			matched = true
		}
	}
	if negated {
		matched = !matched
	}
	return matched, nil
}

func Glob(pattern string) (matches []string, err error) {
	if _, err = Match(pattern, ""); err != nil {
		return nil, err
	}
	if !hasMeta(pattern) {
		if pathExists(pattern) {
			return []string{pattern}, nil
		}
		return nil, nil
	}
	dir, file := Split(pattern)
	if dir == "" {
		dir = "."
	} else {
		dir = Clean(dir)
	}
	if hasMeta(dir) {
		dirs, e := Glob(dir)
		if e != nil {
			return nil, e
		}
		for _, d := range dirs {
			found, e := globDir(d, file)
			if e == nil {
				matches = append(matches, found...)
			}
		}
		return matches, nil
	}
	found, e := globDir(dir, file)
	if e != nil {
		return nil, nil
	}
	return found, nil
}
func globDir(dir, pattern string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, entry := range entries {
		ok, e := Match(pattern, entry.Name())
		if e != nil {
			return nil, e
		}
		if ok {
			out = append(out, Join(dir, entry.Name()))
		}
	}
	return out, nil
}
func pathExists(path string) bool {
	_, err := os.ReadDir(path)
	if err == nil {
		return true
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	f.Close()
	return true
}
func hasMeta(path string) bool {
	for i := 0; i < len(path); i++ {
		if path[i] == '*' || path[i] == '?' || path[i] == '[' {
			return true
		}
	}
	return false
}
func containsSeparator(path string) bool {
	for i := 0; i < len(path); i++ {
		if IsPathSeparator(path[i]) {
			return true
		}
	}
	return false
}
func trimRoot(path string) string {
	for len(path) > 0 && IsPathSeparator(path[0]) {
		path = path[1:]
	}
	return path
}
func splitParts(path string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || IsPathSeparator(path[i]) {
			if i > start {
				out = append(out, path[start:i])
			}
			start = i + 1
		}
	}
	return out
}
func joinParts(parts []string) string {
	out := ""
	for _, part := range parts {
		if out != "" {
			out += "/"
		}
		out += part
	}
	return out
}

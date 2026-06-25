package filepath

import "strings"

const Separator = '/'

func ToSlash(path string) string {
	var out []byte
	i := 0
	for i < len(path) {
		c := path[i]
		if c == '\\' {
			c = '/'
		}
		out = append(out, c)
		i = i + 1
	}
	return string(out)
}

func FromSlash(path string) string {
	return path
}

func IsAbs(path string) bool {
	return len(path) > 0 && path[0] == '/'
}

func Join(a string, b string) string {
	if a == "" {
		return Clean(b)
	}
	if b == "" {
		return Clean(a)
	}
	if strings.HasSuffix(a, "/") {
		return Clean(a + b)
	}
	return Clean(a + "/" + b)
}

func Clean(path string) string {
	if path == "" {
		return "."
	}
	absolute := IsAbs(path)
	parts := strings.Split(ToSlash(path), "/")
	var stack []string
	i := 0
	for i < len(parts) {
		part := parts[i]
		if part == "" || part == "." {
			i = i + 1
		} else if part == ".." {
			if len(stack) > 0 && stack[len(stack)-1] != ".." {
				stack = stack[0 : len(stack)-1]
			} else if !absolute {
				stack = append(stack, part)
			}
			i = i + 1
		} else {
			stack = append(stack, part)
			i = i + 1
		}
	}
	joined := strings.Join(stack, "/")
	if absolute {
		joined = "/" + joined
	}
	if joined == "" {
		if absolute {
			return "/"
		}
		return "."
	}
	return joined
}

func Base(path string) string {
	clean := Clean(path)
	if clean == "/" {
		return "/"
	}
	slash := lastIndexByte(clean, '/')
	if slash < 0 {
		return clean
	}
	return clean[slash+1 : len(clean)]
}

func Dir(path string) string {
	clean := Clean(path)
	if clean == "/" {
		return "/"
	}
	slash := lastIndexByte(clean, '/')
	if slash < 0 {
		return "."
	}
	if slash == 0 {
		return "/"
	}
	return clean[0:slash]
}

func Ext(path string) string {
	base := Base(path)
	i := len(base) - 1
	for i >= 0 {
		if base[i] == '.' {
			return base[i:len(base)]
		}
		i = i - 1
	}
	return ""
}

func lastIndexByte(s string, c byte) int {
	i := len(s) - 1
	for i >= 0 {
		if s[i] == c {
			return i
		}
		i = i - 1
	}
	return -1
}

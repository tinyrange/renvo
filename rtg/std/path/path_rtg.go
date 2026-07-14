//go:build rtg

package path

func IsAbs(path string) bool {
	return len(path) > 0 && path[0] == '/'
}

func Clean(path string) string {
	if path == "" {
		return "."
	}
	out := cleanPath(path)
	if out == "" {
		if IsAbs(path) {
			return "/"
		}
		return "."
	}
	return out
}

func Join(elem ...string) string {
	var out []byte
	for i := 0; i < len(elem); i++ {
		if elem[i] == "" {
			continue
		}
		if len(out) != 0 {
			out = append(out, '/')
		}
		out = appendString(out, elem[i])
	}
	return CleanNoJoin(string(out))
}

func Base(path string) string {
	path = Clean(path)
	if path == "/" {
		return "/"
	}
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

func Dir(path string) string {
	path = Clean(path)
	if path == "/" {
		return "/"
	}
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			if i == 0 {
				return "/"
			}
			return path[:i]
		}
	}
	return "."
}

func Ext(path string) string {
	base := Base(path)
	for i := len(base) - 1; i >= 0; i-- {
		if base[i] == '.' {
			return base[i:]
		}
	}
	return ""
}

func Split(path string) (string, string) {
	i := len(path) - 1
	for i >= 0 && path[i] != '/' {
		i--
	}
	return path[:i+1], path[i+1:]
}

func CleanNoJoin(path string) string {
	return cleanPath(path)
}

func cleanPath(path string) string {
	abs := IsAbs(path)
	parts := splitParts(path)
	var stack []string
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			if len(stack) > 0 && stack[len(stack)-1] != ".." {
				stack = stack[:len(stack)-1]
			} else if !abs {
				stack = append(stack, part)
			}
			continue
		}
		stack = append(stack, part)
	}
	var out []byte
	if abs {
		out = append(out, '/')
	}
	for i := 0; i < len(stack); i++ {
		if i > 0 {
			out = append(out, '/')
		}
		out = appendString(out, stack[i])
	}
	return string(out)
}

func splitParts(path string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '/' {
			out = append(out, path[start:i])
			start = i + 1
		}
	}
	return out
}

func appendString(out []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	return out
}

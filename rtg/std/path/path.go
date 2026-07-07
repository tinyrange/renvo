//go:build !rtg

package path

func IsAbs(path string) bool {
	return len(path) > 0 && path[0] == '/'
}

func Clean(path string) string {
	if path == "" {
		return "."
	}
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
	out := Join(stack...)
	if abs {
		out = prependSlash(out)
	}
	if out == "" {
		if abs {
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

func Split(path string) (dir string, file string) {
	i := len(path) - 1
	for i >= 0 && path[i] != '/' {
		i--
	}
	return path[:i+1], path[i+1:]
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

func CleanNoJoin(path string) string {
	if path == "" {
		return ""
	}
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
	for i := 0; i < len(stack); i++ {
		if i > 0 {
			out = append(out, '/')
		}
		out = appendString(out, stack[i])
	}
	result := string(out)
	if abs {
		result = prependSlash(result)
	}
	return result
}

func appendString(out []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	return out
}

func prependSlash(s string) string {
	out := make([]byte, 0, len(s)+1)
	out = append(out, '/')
	out = appendString(out, s)
	return string(out)
}

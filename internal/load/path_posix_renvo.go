//go:build renvo && !windows

package load

func CleanPath(path string) string {
	if path == "" {
		return "."
	}
	if renvoPathAlreadyClean(path) {
		return path
	}
	absolute := path[0] == '/'
	var out []byte
	rootSize := 0
	if absolute {
		out = append(out, '/')
		rootSize = 1
	}
	i := rootSize
	for i < len(path) {
		for i < len(path) && path[i] == '/' {
			i++
		}
		if i >= len(path) {
			break
		}
		start := i
		for i < len(path) && path[i] != '/' {
			i++
		}
		if i-start == 1 && path[start] == '.' {
			continue
		}
		if i-start == 2 && path[start] == '.' && path[start+1] == '.' {
			last := len(out)
			for last > rootSize && out[last-1] != '/' {
				last--
			}
			pop := len(out) > rootSize && (len(out)-last != 2 || out[last] != '.' || out[last+1] != '.')
			if pop {
				if last > rootSize {
					last--
				}
				out = out[:last]
			} else if !absolute {
				if len(out) > 0 {
					out = append(out, '/')
				}
				out = append(out, '.')
				out = append(out, '.')
			}
			continue
		}
		if len(out) > 0 && out[len(out)-1] != '/' {
			out = append(out, '/')
		}
		for j := start; j < i; j++ {
			out = append(out, path[j])
		}
	}
	if len(out) == 0 {
		return "."
	}
	return string(out)
}

func renvoPathAlreadyClean(path string) bool {
	start := 0
	if path[0] == '/' {
		if len(path) == 1 {
			return true
		}
		start = 1
	}
	for i := start; i <= len(path); i++ {
		if i < len(path) && path[i] != '/' {
			continue
		}
		if i == start || i-start == 1 && path[start] == '.' || i-start == 2 && path[start] == '.' && path[start+1] == '.' {
			return false
		}
		start = i + 1
	}
	return true
}

func JoinPath(base string, elem string) string {
	if elem == "" || elem == "." {
		return CleanPath(base)
	}
	if len(elem) > 0 && elem[0] == '/' {
		return CleanPath(elem)
	}
	if base == "" || base == "." {
		return CleanPath(elem)
	}
	return CleanPath(base + "/" + elem)
}

func RelPath(root string, path string) (string, bool) {
	root = CleanPath(root)
	path = CleanPath(path)
	if root == path {
		return ".", true
	}
	if root == "." && path[0] != '/' {
		if path != ".." && !stringHasPrefix(path, "../") {
			return path, true
		}
		return "", false
	}
	if !hasPathPrefix(path, root) {
		return "", false
	}
	if root == "/" {
		return path[1:], true
	}
	return path[len(root)+1:], true
}

func isAbsPath(path string) bool {
	return len(path) > 0 && path[0] == '/'
}

func isPathArg(arg string) bool {
	if arg == "." || arg == ".." || isAbsPath(arg) {
		return true
	}
	if len(arg) >= 2 && arg[0] == '.' && arg[1] == '/' {
		return true
	}
	return len(arg) >= 3 && arg[0] == '.' && arg[1] == '.' && arg[2] == '/'
}

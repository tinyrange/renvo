//go:build !renvo || windows

package load

func CleanPath(path string) string {
	if path == "" {
		return "."
	}
	driveRoot := pathHasDriveRoot(path)
	absolute := isPathSeparator(path[0]) || driveRoot
	var out []byte
	i := 0
	if driveRoot {
		out = append(out, path[0])
		out = append(out, ':')
		out = append(out, '/')
		i = 3
	} else if absolute {
		out = append(out, '/')
		i = 1
	}
	rootSize := len(out)
	for i < len(path) {
		for i < len(path) && isPathSeparator(path[i]) {
			i++
		}
		if i >= len(path) {
			break
		}
		start := i
		for i < len(path) && !isPathSeparator(path[i]) {
			i++
		}
		if i-start == 1 && path[start] == '.' {
			continue
		}
		if i-start == 2 && path[start] == '.' && path[start+1] == '.' {
			if cleanPathCanPop(out, rootSize) {
				out = cleanPathPop(out, rootSize)
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

func cleanPathCanPop(path []byte, rootSize int) bool {
	if len(path) <= rootSize {
		return false
	}
	start := len(path)
	for start > rootSize && path[start-1] != '/' {
		start--
	}
	return len(path)-start != 2 || path[start] != '.' || path[start+1] != '.'
}

func cleanPathPop(path []byte, rootSize int) []byte {
	start := len(path)
	for start > rootSize && path[start-1] != '/' {
		start--
	}
	if start > rootSize {
		start--
	}
	return path[:start]
}

func JoinPath(base string, elem string) string {
	if elem == "" || elem == "." {
		return CleanPath(base)
	}
	if isAbsPath(elem) {
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
	if root == "." && !isAbsPath(path) {
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
	return len(path) > 0 && (isPathSeparator(path[0]) || pathHasDriveRoot(path))
}

func isPathArg(arg string) bool {
	if arg == "." || arg == ".." || isAbsPath(arg) {
		return true
	}
	if len(arg) >= 2 && arg[0] == '.' && isPathSeparator(arg[1]) {
		return true
	}
	return len(arg) >= 3 && arg[0] == '.' && arg[1] == '.' && isPathSeparator(arg[2])
}

func isPathSeparator(c byte) bool {
	return c == '/' || c == '\\'
}

func pathHasDriveRoot(path string) bool {
	if len(path) < 3 || path[1] != ':' || !isPathSeparator(path[2]) {
		return false
	}
	c := path[0]
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

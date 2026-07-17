//go:build rtg_bundle

package rtg

type StdEntry struct {
	Name  string
	IsDir bool
}

func BundledStdReadFile(path string) ([]byte, bool) {
	name := bundledStdName(path)
	if !bundledStdGoSourceName(name) {
		return nil, false
	}
	data, ok := bundledStdRawReadFile(name)
	if !ok || bundledStdHostOnly(data) {
		return nil, false
	}
	return data, true
}

func BundledStdReadDir(path string) ([]StdEntry, bool) {
	name := bundledStdName(path)
	entries, ok := bundledStdRawReadDir(name)
	if !ok {
		return nil, false
	}
	out := make([]StdEntry, 0, len(entries))
	for i := 0; i < len(entries); i++ {
		entryPath := name + "/" + entries[i].Name
		if entries[i].IsDir {
			if bundledStdDirHasSource(entryPath) {
				out = append(out, entries[i])
			}
			continue
		}
		if !bundledStdGoSourceName(entryPath) {
			continue
		}
		data, readable := bundledStdRawReadFile(entryPath)
		if readable && !bundledStdHostOnly(data) {
			out = append(out, entries[i])
		}
	}
	return out, true
}

func bundledStdDirHasSource(path string) bool {
	entries, ok := bundledStdRawReadDir(path)
	if !ok {
		return false
	}
	for i := 0; i < len(entries); i++ {
		entryPath := path + "/" + entries[i].Name
		if entries[i].IsDir {
			if bundledStdDirHasSource(entryPath) {
				return true
			}
			continue
		}
		if !bundledStdGoSourceName(entryPath) {
			continue
		}
		data, readable := bundledStdRawReadFile(entryPath)
		if readable && !bundledStdHostOnly(data) {
			return true
		}
	}
	return false
}

func bundledStdName(path string) string {
	for len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	return path
}

func bundledStdGoSourceName(path string) bool {
	return bundledStdHasSuffix(path, ".go") && !bundledStdHasSuffix(path, "_test.go")
}

func bundledStdHostOnly(data []byte) bool {
	end := 0
	for end < len(data) && data[end] != '\n' {
		end++
	}
	const constraint = "//go:build !rtg"
	if end != len(constraint) {
		return false
	}
	for i := 0; i < end; i++ {
		if data[i] != constraint[i] {
			return false
		}
	}
	return true
}

func bundledStdHasSuffix(value string, suffix string) bool {
	if len(suffix) > len(value) {
		return false
	}
	start := len(value) - len(suffix)
	for i := 0; i < len(suffix); i++ {
		if value[start+i] != suffix[i] {
			return false
		}
	}
	return true
}

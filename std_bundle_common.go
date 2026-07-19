//go:build renvo_bundle

package renvo

type StdEntry struct {
	Name  string
	IsDir bool
}

func BundledStdReadFile(path string) ([]byte, bool) {
	name, ok := bundledSourceName(path)
	if !ok {
		return nil, false
	}
	if name == "@module/go.mod" {
		return []byte("module renvo.dev\n"), true
	}
	if bundledHasPrefix(name, "@module/") {
		name = name[len("@module/"):]
	}
	if bundledStdHasSuffix(name, "_test.go") {
		return nil, false
	}
	data, ok := bundledStdRawReadFile(name)
	if !ok || bundledStdGoSourceName(name) && bundledStdHostOnly(data) {
		return nil, false
	}
	return data, true
}

func BundledStdReadDir(path string) ([]StdEntry, bool) {
	name, ok := bundledSourceName(path)
	if !ok {
		return nil, false
	}
	if name == "@module" {
		return []StdEntry{{Name: "go.mod"}, {Name: "forms", IsDir: true}, {Name: "std", IsDir: true}}, true
	}
	if bundledHasPrefix(name, "@module/") {
		name = name[len("@module/"):]
	}
	entries, ok := bundledStdRawReadDir(name)
	if !ok {
		return nil, false
	}
	out := make([]StdEntry, 0, len(entries))
	for i := 0; i < len(entries); i++ {
		entryPath := name + "/" + entries[i].Name
		if entries[i].IsDir {
			out = append(out, entries[i])
			continue
		}
		if bundledStdHasSuffix(entryPath, "_test.go") {
			continue
		}
		if bundledStdGoSourceName(entryPath) {
			data, readable := bundledStdRawReadFile(entryPath)
			if !readable || bundledStdHostOnly(data) {
				continue
			}
		}
		out = append(out, entries[i])
	}
	return out, true
}

func bundledSourceName(path string) (string, bool) {
	for len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	const module = "modules/renvo.dev@v0.0.0"
	if path == module {
		return "@module", true
	}
	if bundledHasPrefix(path, module+"/") {
		return "@module/" + path[len(module)+1:], true
	}
	if path != "std" && !bundledHasPrefix(path, "std/") {
		return "", false
	}
	return path, true
}

func bundledHasPrefix(value string, prefix string) bool {
	if len(value) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if value[i] != prefix[i] {
			return false
		}
	}
	return true
}

func bundledStdGoSourceName(path string) bool {
	return bundledStdHasSuffix(path, ".go") && !bundledStdHasSuffix(path, "_test.go")
}

func bundledStdHostOnly(data []byte) bool {
	end := 0
	for end < len(data) && data[end] != '\n' {
		end++
	}
	const constraint = "//go:build !renvo"
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

//go:build renvo_bundle && !renvo

package renvo

import "embed"

//go:embed std
var bundledStdFiles embed.FS

func bundledStdRawReadFile(path string) ([]byte, bool) {
	data, err := bundledStdFiles.ReadFile(path)
	return data, err == nil
}

func bundledStdRawReadDir(path string) ([]StdEntry, bool) {
	entries, err := bundledStdFiles.ReadDir(path)
	if err != nil {
		return nil, false
	}
	out := make([]StdEntry, 0, len(entries))
	for i := 0; i < len(entries); i++ {
		out = append(out, StdEntry{Name: entries[i].Name(), IsDir: entries[i].IsDir()})
	}
	return out, true
}

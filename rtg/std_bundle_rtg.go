//go:build rtg_bundle && rtg

package rtg

import "embed"

//go:embed std
var bundledStdFiles embed.FS

func bundledStdRawReadFile(path string) ([]byte, bool) {
	return bundledStdFiles.ReadFileOK(path)
}

func bundledStdRawReadDir(path string) ([]StdEntry, bool) {
	entries, ok := bundledStdFiles.ReadDirOK(path)
	if !ok {
		return nil, false
	}
	out := make([]StdEntry, 0, len(entries))
	for i := 0; i < len(entries); i++ {
		out = append(out, StdEntry{Name: entries[i].Name(), IsDir: entries[i].IsDir()})
	}
	return out, true
}

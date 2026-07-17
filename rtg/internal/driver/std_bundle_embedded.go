//go:build rtg_bundle

package driver

import stdbundle "j5.nz/rtg/rtg"

const rtgBundledStdEnabled = true

func bundledStdReadFile(path string) ([]byte, bool) {
	return stdbundle.BundledStdReadFile(path)
}

func bundledStdReadDir(path string) ([]DirEntry, bool) {
	entries, ok := stdbundle.BundledStdReadDir(path)
	if !ok {
		return nil, false
	}
	out := make([]DirEntry, 0, len(entries))
	for i := 0; i < len(entries); i++ {
		out = append(out, DirEntry{Name: entries[i].Name, IsDir: entries[i].IsDir})
	}
	return out, true
}

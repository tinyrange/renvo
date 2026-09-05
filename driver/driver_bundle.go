//go:build renvo_bundle

package driver

import (
	"renvo.dev"
)

type bundledSourceFS struct{}

func (b *bundledSourceFS) PathExists(path string) bool {
	_, ok := renvo.BundledStdReadFile(path)
	return ok
}

func (b *bundledSourceFS) ReadDir(path string) ([]DirEntry, bool) {
	results, ok := renvo.BundledStdReadDir(path)

	var ret []DirEntry

	for _, entry := range results {
		ret = append(ret, DirEntry(entry))
	}

	return ret, ok
}

func (b *bundledSourceFS) ReadFile(path string) ([]byte, bool) {
	// Implement the logic to read a file from the bundled source filesystem.
	return renvo.BundledStdReadFile(path)
}

var (
	_ SourceFS = &bundledSourceFS{}
)

func BundledSourceFS() SourceFS {
	return &bundledSourceFS{}
}

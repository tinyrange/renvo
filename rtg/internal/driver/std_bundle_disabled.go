//go:build !rtg_bundle

package driver

const rtgBundledStdEnabled = false

func bundledStdReadFile(path string) ([]byte, bool) {
	return nil, false
}

func bundledStdReadDir(path string) ([]DirEntry, bool) {
	return nil, false
}

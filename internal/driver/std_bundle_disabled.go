//go:build !renvo_bundle

package driver

const renvoBundledStdEnabled = false

func bundledStdReadFile(path string) ([]byte, bool) {
	return nil, false
}

func bundledStdReadDir(path string) ([]DirEntry, bool) {
	return nil, false
}

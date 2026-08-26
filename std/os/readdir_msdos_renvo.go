//go:build renvo && msdos

package os

// DOS directory enumeration uses FindFirst/FindNext rather than file reads.
// Applications that need it use renvo.dev/device/dos.Find; importing the full
// device surface here would consume too much of a 16-bit program segment.
func ReadDir(name string) ([]DirEntry, error) {
	_ = name
	return nil, errIO()
}

//go:build renvo && msdos

package os

// DOS directory enumeration uses FindFirst/FindNext rather than file reads.
// The minimal backend syscall contract does not expose those services, so keep
// ReadDir available to portable packages and report the operation cleanly.
func ReadDir(name string) ([]DirEntry, error) {
	_ = name
	return nil, errIO()
}

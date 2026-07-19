//go:build renvo && windows

package os

// renvo:linkstatic kernel32.dll,GetCurrentDirectoryA
func renvoWindowsGetCurrentDirectory(size int, buffer *byte) int { return 0 }

func Getwd() (string, error) {
	buf := make([]byte, 32768)
	n := renvoWindowsGetCurrentDirectory(len(buf), &buf[0])
	if n <= 0 || n >= len(buf) {
		return "", errIO()
	}
	return string(buf[:n]), nil
}

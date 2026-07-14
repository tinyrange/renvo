//go:build rtg && windows

package os

// rtg:linkstatic kernel32.dll,GetCurrentDirectoryA
func rtgWindowsGetCurrentDirectory(size int, buffer *byte) int { return 0 }

func Getwd() (string, *osError) {
	buf := make([]byte, 32768)
	n := rtgWindowsGetCurrentDirectory(len(buf), &buf[0])
	if n <= 0 || n >= len(buf) {
		return "", errIO()
	}
	return string(buf[:n]), nil
}

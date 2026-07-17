//go:build rtg && windows

package process

// WinExec is available on the old and new Windows versions supported by the
// graphics backend, including Windows 98.
// rtg:linkstatic kernel32.dll,WinExec
func winExec(command []byte, show int) int { return 0 }

func Start(path, directory string) bool {
	command := make([]byte, 0, len(path)+3)
	command = append(command, '"')
	command = append(command, path...)
	command = append(command, '"', 0)
	return winExec(command, 1) > 31
}

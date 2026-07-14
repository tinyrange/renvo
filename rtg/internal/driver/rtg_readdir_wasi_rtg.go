//go:build rtg && (wasi || wasip1)

package driver

func rtgReadDirNative(path string) ([]DirEntry, bool) {
	fd := open(rtgPathCString(path), 0)
	if fd < 0 {
		return nil, false
	}
	buf := make([]byte, 32768)
	n := syscall(rtgGetdents64LinuxAmd64, fd, buf, len(buf))
	close(fd)
	if n < 0 {
		return nil, false
	}
	out, ok := rtgAppendDirentBuffer(nil, buf, n)
	if !ok {
		return nil, false
	}
	sortDirEntries(out)
	return out, true
}

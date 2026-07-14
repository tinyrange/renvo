//go:build rtg && linux && 386

package driver

func rtgReadDirNative(path string) ([]DirEntry, bool) {
	fd := open(rtgPathCString(path), 0)
	if fd < 0 {
		return nil, false
	}
	buf := make([]byte, 32768)
	out := make([]DirEntry, 0, 32)
	for {
		n := syscall(rtgGetdents64Linux386, fd, buf, len(buf))
		if n < 0 {
			close(fd)
			return nil, false
		}
		if n == 0 {
			break
		}
		pos := 0
		minimum := rtgDirentMinimum()
		for pos+minimum <= n {
			reclen := rtgDirentRecordLength(buf, pos)
			if reclen <= minimum || pos+reclen > n {
				close(fd)
				return nil, false
			}
			nameStart := rtgDirentNameStart(pos)
			typeAt := rtgDirentTypeOffset(pos)
			nameEnd := nameStart
			for nameEnd < pos+reclen && buf[nameEnd] != 0 {
				nameEnd++
			}
			if nameEnd > nameStart && !rtgDirNameIsDot(buf, nameStart, nameEnd) {
				out = append(out, DirEntry{Name: string(buf[nameStart:nameEnd]), IsDir: buf[typeAt] == 4})
			}
			pos += reclen
		}
	}
	close(fd)
	sortDirEntries(out)
	return out, true
}

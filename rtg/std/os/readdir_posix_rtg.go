//go:build rtg && !windows && !wasi && !wasip1

package os

func ReadDir(name string) ([]DirEntry, *osError) {
	fd := open(rtgPathCString(name), O_RDONLY)
	if fd < 0 {
		return nil, errIO()
	}
	buf := make([]byte, 32768)
	var out []DirEntry
	for {
		n := rtgReadDirChunk(fd, buf)
		if n < 0 {
			close(fd)
			return nil, errIO()
		}
		if n == 0 {
			break
		}
		var ok bool
		out, ok = rtgAppendDirentBuffer(out, buf, n)
		if !ok {
			close(fd)
			return nil, errIO()
		}
	}
	close(fd)
	sortDirEntries(out)
	return out, nil
}

func rtgAppendDirentBuffer(out []DirEntry, buf []byte, n int) ([]DirEntry, bool) {
	pos := 0
	minimum := rtgDirentMinimum()
	for pos+minimum <= n {
		reclen := rtgDirentRecordLength(buf, pos)
		if reclen <= minimum || pos+reclen > n {
			return out, false
		}
		typeAt := rtgDirentTypeOffset(pos)
		nameStart := rtgDirentNameStart(pos)
		nameEnd := nameStart
		for nameEnd < pos+reclen && buf[nameEnd] != 0 {
			nameEnd++
		}
		if nameEnd > nameStart && !dirNameIsDot(buf, nameStart, nameEnd) {
			out = append(out, DirEntry{name: string(buf[nameStart:nameEnd]), isDir: rtgDirentIsDirectory(buf[typeAt])})
		}
		pos += reclen
	}
	return out, true
}

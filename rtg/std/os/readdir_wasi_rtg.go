//go:build rtg && (wasi || wasip1)

package os

// The wasm backend recognizes this intrinsic selector and lowers it to the
// WASI fd_readdir import.
const rtgFdReaddirIntrinsic = 217

func ReadDir(name string) ([]DirEntry, error) {
	fd := open(rtgPathCString(name), O_RDONLY)
	if fd < 0 {
		return nil, errIO()
	}
	buf := make([]byte, 32768)
	n := syscall(rtgFdReaddirIntrinsic, fd, buf, len(buf))
	close(fd)
	if n < 0 {
		return nil, errIO()
	}
	out, ok := rtgAppendWasiDirentBuffer(nil, buf, n)
	if !ok {
		return nil, errIO()
	}
	sortDirEntries(out)
	return out, nil
}

func rtgAppendWasiDirentBuffer(out []DirEntry, buf []byte, n int) ([]DirEntry, bool) {
	pos := 0
	minimum := rtgDirentMinimum()
	for pos+minimum <= n {
		reclen := rtgDirentRecordLength(buf, pos)
		if reclen <= minimum || pos+reclen > n {
			return out, false
		}
		nameStart := rtgDirentNameStart(pos)
		nameEnd := pos + reclen
		if nameEnd > nameStart && !dirNameIsDot(buf, nameStart, nameEnd) {
			out = append(out, DirEntry{name: string(buf[nameStart:nameEnd]), isDir: rtgDirentIsDirectory(buf[rtgDirentTypeOffset(pos)])})
		}
		pos += reclen
	}
	return out, true
}

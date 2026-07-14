//go:build rtg && (wasi || wasip1)

package driver

func rtgAppendDirentBuffer(out []DirEntry, buf []byte, n int) ([]DirEntry, bool) {
	pos := 0
	minimum := rtgDirentMinimum()
	for pos+minimum <= n {
		reclen := rtgDirentRecordLength(buf, pos)
		if reclen <= minimum || pos+reclen > n {
			return out, false
		}
		nameStart := rtgDirentNameStart(pos)
		typeAt := rtgDirentTypeOffset(pos)
		nameEnd := nameStart
		for nameEnd < pos+reclen && buf[nameEnd] != 0 {
			nameEnd++
		}
		if nameEnd > nameStart && !rtgDirNameIsDot(buf, nameStart, nameEnd) {
			out = append(out, DirEntry{Name: string(buf[nameStart:nameEnd]), IsDir: rtgDirentIsDirectory(buf[typeAt])})
		}
		pos += reclen
	}
	return out, true
}

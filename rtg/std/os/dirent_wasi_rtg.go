//go:build rtg && (wasi || wasip1)

package os

func rtgDirentMinimum() int { return 24 }

func rtgDirentRecordLength(buf []byte, pos int) int {
	return 24 + int(buf[pos+16]) + int(buf[pos+17])<<8 + int(buf[pos+18])<<16 + int(buf[pos+19])<<24
}

func rtgDirentTypeOffset(pos int) int { return pos + 20 }

func rtgDirentNameStart(pos int) int { return pos + 24 }

func rtgDirentIsDirectory(typ byte) bool { return typ == 3 }

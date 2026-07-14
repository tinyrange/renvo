//go:build rtg && !darwin && !wasi && !wasip1

package os

func rtgDirentMinimum() int { return 19 }

func rtgDirentRecordLength(buf []byte, pos int) int {
	return int(buf[pos+16]) | int(buf[pos+17])<<8
}

func rtgDirentTypeOffset(pos int) int { return pos + 18 }

func rtgDirentNameStart(pos int) int { return pos + 19 }

func rtgDirentIsDirectory(typ byte) bool { return typ == 4 }

//go:build rtg && darwin && arm64

package os

func rtgDirentMinimum() int { return 8 }

func rtgDirentRecordLength(buf []byte, pos int) int {
	return int(buf[pos+4]) | int(buf[pos+5])<<8
}

func rtgDirentTypeOffset(pos int) int { return pos + 6 }

func rtgDirentNameStart(pos int) int { return pos + 8 }

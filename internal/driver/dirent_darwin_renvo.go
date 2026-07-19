//go:build renvo && darwin && arm64

package driver

func renvoDirentMinimum() int { return 8 }

func renvoDirentRecordLength(buf []byte, pos int) int {
	return int(buf[pos+4]) | int(buf[pos+5])<<8
}

func renvoDirentTypeOffset(pos int) int { return pos + 6 }

func renvoDirentNameStart(pos int) int { return pos + 8 }

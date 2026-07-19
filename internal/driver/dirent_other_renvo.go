//go:build renvo && !darwin && !wasi && !wasip1

package driver

func renvoDirentMinimum() int { return 19 }

func renvoDirentRecordLength(buf []byte, pos int) int {
	return int(buf[pos+16]) | int(buf[pos+17])<<8
}

func renvoDirentTypeOffset(pos int) int { return pos + 18 }

func renvoDirentNameStart(pos int) int { return pos + 19 }

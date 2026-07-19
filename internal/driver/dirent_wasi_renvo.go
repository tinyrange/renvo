//go:build renvo && (wasi || wasip1)

package driver

func renvoDirentMinimum() int { return 24 }

func renvoDirentRecordLength(buf []byte, pos int) int {
	return 24 + int(buf[pos+16]) + int(buf[pos+17])<<8 + int(buf[pos+18])<<16 + int(buf[pos+19])<<24
}

func renvoDirentTypeOffset(pos int) int { return pos + 20 }

func renvoDirentNameStart(pos int) int { return pos + 24 }

func renvoDirentIsDirectory(typ byte) bool { return typ == 3 }

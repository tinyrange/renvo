//go:build renvo && netbsd && amd64

package os

func renvoDirentMinimum() int { return 13 }

func renvoDirentRecordLength(buf []byte, pos int) int {
	return int(buf[pos+8]) | int(buf[pos+9])<<8
}

func renvoDirentTypeOffset(pos int) int { return pos + 12 }

func renvoDirentNameStart(pos int) int { return pos + 13 }

func renvoDirentIsDirectory(typ byte) bool { return typ == 4 }

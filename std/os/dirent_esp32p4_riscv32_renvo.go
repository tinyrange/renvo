//go:build renvo && esp32p4 && riscv32

package os

// The ESP32-P4 compiler host presents directory entries in the compact
// Linux dirent64 layout so ReadDir does not need a target-specific syscall.
func renvoDirentMinimum() int { return 19 }

func renvoDirentRecordLength(buf []byte, pos int) int {
	return int(buf[pos+16]) | int(buf[pos+17])<<8
}

func renvoDirentTypeOffset(pos int) int { return pos + 18 }

func renvoDirentNameStart(pos int) int { return pos + 19 }

func renvoDirentIsDirectory(typ byte) bool { return typ == 4 }

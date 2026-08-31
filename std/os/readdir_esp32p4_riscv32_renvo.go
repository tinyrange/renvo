//go:build renvo && esp32p4 && riscv32

package os

func renvoReadDirChunk(fd int, buf []byte) int {
	return read(fd, buf, -1)
}

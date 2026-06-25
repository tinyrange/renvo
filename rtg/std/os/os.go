package os

const O_RDONLY = 0
const O_WRONLY = 1
const O_RDWR = 2
const O_CREATE = 64
const O_TRUNC = 512

const Stdin = 0
const Stdout = 1
const Stderr = 2

func Open(path string, flags int) int {
	return -1
}

func Close(fd int) int {
	return -1
}

func Read(fd int, buf []byte, off int64) int {
	return -1
}

func Write(fd int, buf []byte, off int64) int {
	return -1
}

func Chmod(fd int, mode int) int {
	return -1
}

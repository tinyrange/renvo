//go:build renvo && freebsd && amd64

package os

func syscall(num int, fd int, buf []byte, size int, base int) int { return 0 }

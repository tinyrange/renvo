//go:build renvo && !windows

package os

// syscall is a compiler intrinsic. Target-selected adapters are the only
// callers, so a binary can never probe another platform's syscall table.
func syscall(num int, fd int, buf []byte, size int) int { return 0 }

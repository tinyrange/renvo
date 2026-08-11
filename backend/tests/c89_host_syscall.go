package main

func syscall(number int, fd int, buffer []byte, size int) int { return -1 }

func appMain() int {
	buffer := make([]byte, 32)
	if syscall(217, -1, buffer, len(buffer)) >= 0 {
		return 1
	}
	print("PASS\n")
	return 0
}

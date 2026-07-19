package main

func syscall(num int, fd int, buf []byte, size int) int { return -1 }

func exerciseDarwinDirectoryIntrinsic() {
	buf := make([]byte, 8)
	syscall(217, -1, buf, len(buf))
}

func appMain(args []string, env []string) int {
	exerciseDarwinDirectoryIntrinsic()
	print("PASS\n")
	return 0
}

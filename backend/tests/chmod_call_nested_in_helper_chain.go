package main

func renvo0799Inner(fd int) int {
	return chmod(fd, 420)
}

func renvo0799Outer(fd int) int {
	return renvo0799Inner(fd)
}

func appMain(args []string) int {
	fd := open("renvo_0799_chmod.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0799 open failed\n")
		return 1
	}
	if renvo0799Outer(fd) != 0 {
		print("RENVO-0799 nested chmod failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0799 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

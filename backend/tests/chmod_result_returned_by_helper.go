package main

func renvo0788Do(fd int) int {
	return chmod(fd, 420)
}

func appMain(args []string) int {
	fd := open("renvo_0788_chmod.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0788 open failed\n")
		return 1
	}
	if renvo0788Do(fd) != 0 {
		print("RENVO-0788 helper chmod failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0788 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

package main

func appMain(args []string) int {
	fd := open("renvo_0795_chmod.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0795 open failed\n")
		return 1
	}
	b := []byte("z")
	if chmod(fd, 420) != 0 {
		print("RENVO-0795 chmod failed\n")
		return 1
	}
	if write(fd, b, 0) != 1 {
		print("RENVO-0795 descriptor write after chmod failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0795 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

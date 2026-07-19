package main

func appMain(args []string) int {
	fd := open("renvo_0763_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0763 open failed\n")
		return 1
	}
	b := []byte("abcd")
	if write(fd, b, 0) != 4 {
		print("RENVO-0763 seed write failed\n")
		return 1
	}
	r := []byte("ab")
	if read(fd, r, 0) != 2 {
		print("RENVO-0763 bounded read failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0763 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

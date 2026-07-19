package main

func appMain(args []string) int {
	fd := open("renvo_0757_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0757 open failed\n")
		return 1
	}
	b := []byte("abcd")
	if write(fd, b, 0) != 4 {
		print("RENVO-0757 seed write failed\n")
		return 1
	}
	r := []byte(" ")
	if read(fd, r, 0) != 1 {
		print("RENVO-0757 read failed\n")
		return 1
	}
	if r[0] != 'a' {
		print("RENVO-0757 value failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0757 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

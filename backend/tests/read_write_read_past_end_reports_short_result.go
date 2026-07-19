package main

func appMain(args []string) int {
	fd := open("renvo_0764_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0764 open failed\n")
		return 1
	}
	b := []byte("abcd")
	if write(fd, b, 0) != 4 {
		print("RENVO-0764 seed write failed\n")
		return 1
	}
	r := []byte("d")
	if read(fd, r, 3) != 1 {
		print("RENVO-0764 end read failed\n")
		return 1
	}
	if r[0] != 'd' {
		print("RENVO-0764 value failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0764 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

package main

func appMain(args []string) int {
	fd := open("renvo_0740_open.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0740 open failed\n")
		return 1
	}
	b := []byte("q")
	if write(fd, b, 0) != 1 {
		print("RENVO-0740 write failed\n")
		return 1
	}
	r := []byte(" ")
	if read(fd, r, 0) != 1 {
		print("RENVO-0740 read failed\n")
		return 1
	}
	if r[0] != 'q' {
		print("RENVO-0740 read value failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0740 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

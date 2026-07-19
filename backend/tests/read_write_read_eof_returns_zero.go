package main

func appMain(args []string) int {
	fd := open("renvo_0820_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0820 open failed\n")
		return 1
	}
	data := []byte("abcd")
	if write(fd, data, 0) != 4 {
		print("RENVO-0820 seed write failed\n")
		return 1
	}
	wide := []byte("xxxx")
	if read(fd, wide, 2) != 2 {
		print("RENVO-0820 partial eof read failed\n")
		return 1
	}
	if wide[0] != 'c' {
		print("RENVO-0820 partial first byte failed\n")
		return 1
	}
	if wide[1] != 'd' {
		print("RENVO-0820 partial second byte failed\n")
		return 1
	}
	empty := []byte("z")
	if read(fd, empty, 4) != 0 {
		print("RENVO-0820 eof read failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0820 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

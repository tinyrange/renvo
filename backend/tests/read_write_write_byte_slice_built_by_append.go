package main

func appMain(args []string) int {
	fd := open("renvo_0754_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0754 open failed\n")
		return 1
	}
	var b []byte
	b = append(b, 'h')
	b = append(b, 'i')
	if write(fd, b, 0) != 2 {
		print("RENVO-0754 append write failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0754 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

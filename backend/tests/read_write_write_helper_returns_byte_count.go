package main

func renvo0767Write(fd int, b []byte) int {
	return write(fd, b, 0)
}

func appMain(args []string) int {
	fd := open("renvo_0767_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0767 open failed\n")
		return 1
	}
	b := []byte("xy")
	if renvo0767Write(fd, b) != 2 {
		print("RENVO-0767 helper write failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0767 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

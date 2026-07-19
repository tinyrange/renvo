package main

func renvo0768Read(fd int, b []byte) int {
	return read(fd, b, 0)
}

func appMain(args []string) int {
	fd := open("renvo_0768_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0768 open failed\n")
		return 1
	}
	b := []byte("abcd")
	if write(fd, b, 0) != 4 {
		print("RENVO-0768 seed write failed\n")
		return 1
	}
	r := []byte("    ")
	if renvo0768Read(fd, r) != 4 {
		print("RENVO-0768 helper read failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0768 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

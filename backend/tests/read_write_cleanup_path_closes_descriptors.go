package main

func appMain(args []string) int {
	fd := open("renvo_0775_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0775 open failed\n")
		return 1
	}
	b := []byte("m")
	if write(fd, b, 0) != 1 {
		print("RENVO-0775 write failed\n")
		return 1
	}
	goto cleanup
cleanup:
	if close(fd) != 0 {
		print("RENVO-0775 cleanup close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

package main

func appMain(args []string) int {
	fd := open("renvo_0752_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0752 open failed\n")
		return 1
	}
	b := []byte("ABC")
	if write(fd, b, 0) != 3 {
		print("RENVO-0752 write failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0752 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

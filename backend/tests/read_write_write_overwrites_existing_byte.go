package main

func appMain(args []string) int {
	fd := open("renvo_0756_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0756 open failed\n")
		return 1
	}
	b := []byte("ab")
	if write(fd, b, 0) != 2 {
		print("RENVO-0756 initial write failed\n")
		return 1
	}
	c := []byte("Z")
	if write(fd, c, 1) != 1 {
		print("RENVO-0756 overwrite failed\n")
		return 1
	}
	r := []byte("  ")
	if read(fd, r, 0) != 2 {
		print("RENVO-0756 read failed\n")
		return 1
	}
	if r[1] != 'Z' {
		print("RENVO-0756 overwrite value failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0756 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

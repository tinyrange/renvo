package main

func appMain(args []string) int {
	fd := open("renvo_0772_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0772 open failed\n")
		return 1
	}
	b := []byte("abcd")
	if write(fd, b, 0) != 4 {
		print("RENVO-0772 seed write failed\n")
		return 1
	}
	i := 0
	for i < 4 {
		r := []byte(" ")
		if read(fd, r, int64(i)) != 1 {
			print("RENVO-0772 loop read failed\n")
			return 2
		}
		if int(r[0]) != 97+i {
			print("RENVO-0772 loop value failed\n")
			return 3
		}
		i = i + 1
	}
	if close(fd) != 0 {
		print("RENVO-0772 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

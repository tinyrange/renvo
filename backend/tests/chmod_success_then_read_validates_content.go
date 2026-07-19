package main

func appMain(args []string) int {
	fd := open("renvo_0797_chmod.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0797 open failed\n")
		return 1
	}
	b := []byte("r")
	if write(fd, b, 0) != 1 {
		print("RENVO-0797 seed write failed\n")
		return 1
	}
	if chmod(fd, 420) != 0 {
		print("RENVO-0797 chmod failed\n")
		return 1
	}
	r := []byte(" ")
	if read(fd, r, 0) != 1 {
		print("RENVO-0797 read after chmod failed\n")
		return 1
	}
	if r[0] != 'r' {
		print("RENVO-0797 read value failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0797 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

package main

func appMain(args []string) int {
	fd := open("renvo_0766_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0766 open failed\n")
		return 1
	}
	var b []byte
	if read(fd, b, 0) != 0 {
		print("RENVO-0766 empty read failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0766 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

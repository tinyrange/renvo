package main

type renvo0774Box struct {
	buf []byte
}

func appMain(args []string) int {
	fd := open("renvo_0774_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0774 open failed\n")
		return 1
	}
	b := []byte("abcd")
	if write(fd, b, 0) != 4 {
		print("RENVO-0774 seed write failed\n")
		return 1
	}
	box := renvo0774Box{buf: []byte("    ")}
	if read(fd, box.buf, 0) != 4 {
		print("RENVO-0774 boxed read failed\n")
		return 1
	}
	if box.buf[1] != 'b' {
		print("RENVO-0774 boxed read value failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0774 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

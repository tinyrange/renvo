package main

type renvo0773Box struct {
	buf []byte
}

func appMain(args []string) int {
	fd := open("renvo_0773_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0773 open failed\n")
		return 1
	}
	box := renvo0773Box{buf: []byte("pq")}
	if write(fd, box.buf, 0) != 2 {
		print("RENVO-0773 boxed write failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0773 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

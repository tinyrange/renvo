package main

func appMain(args []string) int {
	flags := O_RDWR
	flags = flags | O_CREATE
	flags = flags | O_TRUNC
	fd := open("renvo_0736_open.tmp", flags)
	if fd < 0 {
		print("RENVO-0736 open failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0736 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

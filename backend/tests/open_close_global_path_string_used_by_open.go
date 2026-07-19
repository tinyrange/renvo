package main

var renvo0746Path string = "renvo_0746_open.tmp"

func appMain(args []string) int {
	fd := open(renvo0746Path, O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0746 open failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0746 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

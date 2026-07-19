package main

func renvo0734Path() string {
	return "renvo_0734_open.tmp"
}

func appMain(args []string) int {
	fd := open(renvo0734Path(), O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0734 open failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0734 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

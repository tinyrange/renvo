package main

func appMain(args []string) int {
	a := open("renvo_0738_open.tmp", O_RDWR|O_CREATE|O_TRUNC)
	b := open("renvo_0738_open.tmp", O_RDWR)
	if a < 0 {
		print("RENVO-0738 open a failed\n")
		return 1
	}
	if b < 0 {
		print("RENVO-0738 open b failed\n")
		return 1
	}
	if a == b {
		print("RENVO-0738 descriptors not separate\n")
		return 1
	}
	if close(a) != 0 {
		print("RENVO-0738 close a failed\n")
		return 1
	}
	if close(b) != 0 {
		print("RENVO-0738 close b failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

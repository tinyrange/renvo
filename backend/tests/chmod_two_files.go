package main

func appMain(args []string) int {
	a := open("renvo_0793_chmod.tmpa", O_RDWR|O_CREATE|O_TRUNC)
	b := open("renvo_0793_chmod.tmpb", O_RDWR|O_CREATE|O_TRUNC)
	if a < 0 {
		print("RENVO-0793 open a failed\n")
		return 1
	}
	if b < 0 {
		print("RENVO-0793 open b failed\n")
		return 1
	}
	if chmod(a, 420) != 0 {
		print("RENVO-0793 chmod a failed\n")
		return 1
	}
	if chmod(b, 384) != 0 {
		print("RENVO-0793 chmod b failed\n")
		return 1
	}
	if close(a) != 0 {
		print("RENVO-0793 close a failed\n")
		return 1
	}
	if close(b) != 0 {
		print("RENVO-0793 close b failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

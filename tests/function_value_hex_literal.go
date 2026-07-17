package main

func appMain() int {
	fn := func() uint64 { return 0xe9b5dba5 }
	if fn() == 0xe9b5dba5 {
		print("PASS\n")
		return 0
	}
	return 1
}

package main

type renvoIncdecCounter struct {
	count int
}

func appMain(args []string) int {
	counter := renvoIncdecCounter{count: 2}
	counter.count++
	counter.count++
	if counter.count != 4 {
		print("RENVO-INCDEC-008 struct field increment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

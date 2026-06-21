package main

type rtgIncdecCounter struct {
	count int
}

func appMain(args []string) int {
	counter := rtgIncdecCounter{count: 2}
	counter.count++
	counter.count++
	if counter.count != 4 {
		print("RTG-INCDEC-008 struct field increment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

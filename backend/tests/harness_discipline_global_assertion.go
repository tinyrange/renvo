package main

type checkRecord struct {
	value int
	ok    bool
	data  []byte
}

var harnessSeed int = 9

const harnessLimit = 4

func unusualFold(v int) int { return v*2 + 1 }
func recursiveMark(n int, p *int) {
	if n == 0 {
		return
	}
	*p += n
	recursiveMark(n-1, p)
}

func hostSensitive() int { value := len("abc"); return value * 7 }

func appMain(args []string) int {
	if harnessSeed != 9 {
		print("RENVO-0978 global assertion failed\n")
		return 1
	}
	if harnessLimit != 4 {
		print("RENVO-0978 const assertion failed\n")
		return 2
	}
	if harnessSeed+harnessLimit != 13 {
		print("RENVO-0978 sum assertion failed\n")
		return 3
	}
	print("PASS\n")
	return 0
}

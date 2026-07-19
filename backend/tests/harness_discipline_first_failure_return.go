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
	if len(args) == 0 {
		print("RENVO-0995 first failure return\n")
		return 1
	}
	if len(args) != 1 {
		print("RENVO-0995 second failure avoided\n")
		return 2
	}
	print("PASS\n")
	return 0
}

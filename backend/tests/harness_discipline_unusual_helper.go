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
	for i := 0; i < 3; i += 1 {
		if unusualFold(i) == 99 {
			print("RENVO-0984 unusual helper failed\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}

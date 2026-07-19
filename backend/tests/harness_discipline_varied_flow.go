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
	value := 0
	i := 0
	for i < 5 {
		i += 1
		if i == 3 {
			continue
		}
		value += i
	}
	if value != 12 {
		print("RENVO-0986 varied flow failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

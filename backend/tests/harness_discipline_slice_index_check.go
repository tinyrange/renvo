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
	var r checkRecord
	r.data = append(r.data, 'x')
	p := &r
	if p.data[0] != 'x' {
		print("RENVO-0990 slice index check failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

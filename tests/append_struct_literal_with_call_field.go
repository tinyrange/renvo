package main

type appendCallFieldRec struct {
	kind  int
	start int
	end   int
	line  int
}

func classifyCallField(src []byte, start int, end int) int {
	if end-start == 4 && src[start] == 'f' {
		return 10
	}
	return 1
}

func scanCallField(src []byte) []appendCallFieldRec {
	out := make([]appendCallFieldRec, 0, 128)
	i := 0
	line := 1
	for i < len(src) {
		start := i
		for i < len(src) && src[i] != ' ' {
			i++
		}
		out = append(out, appendCallFieldRec{kind: classifyCallField(src, start, i), start: start, end: i, line: line})
		if i < len(src) {
			i++
		}
	}
	return out
}

func appMain() int {
	src := []byte("package main func appMain")
	out := scanCallField(src)
	if len(out) == 4 && out[0].kind == 1 && out[1].start == 8 && out[2].kind == 10 && out[2].start == 13 && out[3].end == len(src) {
		print("PASS\n")
	}
	return 0
}

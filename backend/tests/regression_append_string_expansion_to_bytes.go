package main

type token struct {
	Text string
}

func build(tokens []token) []byte {
	var out []byte
	prefix := "PA"
	out = append(out, prefix...)
	for i := 0; i < len(tokens); i++ {
		text := tokens[i].Text
		out = append(out, text...)
	}
	return out
}

func appMain() int {
	tokens := []token{{Text: "SS\n"}}
	got := build(tokens)
	if len(got) != 5 {
		print("FAIL\n")
		return 1
	}
	if got[0] != 'P' || got[1] != 'A' || got[2] != 'S' || got[3] != 'S' || got[4] != '\n' {
		print("FAIL\n")
		return 1
	}
	print(string(got))
	return 0
}

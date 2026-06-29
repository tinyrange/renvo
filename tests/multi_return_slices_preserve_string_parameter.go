package main

type preserveTemp struct {
	name string
	expr string
}

type preserveReplacement struct {
	start int
	end   int
	text  string
}

func preserveBuild() ([]preserveTemp, []preserveReplacement) {
	var temps []preserveTemp
	var replacements []preserveReplacement
	temps = append(temps, preserveTemp{name: "alpha", expr: "beta"})
	replacements = append(replacements, preserveReplacement{start: 1, end: 3, text: "X"})
	return temps, replacements
}

func preserveAppendString(out []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	return out
}

func preserveBody(body string) string {
	temps, replacements := preserveBuild()
	var out []byte
	out = preserveAppendString(out, body[0:6])
	for i := 0; i < len(temps); i++ {
		temp := temps[i]
		out = preserveAppendString(out, temp.name)
		out = preserveAppendString(out, temp.expr)
	}
	for i := 0; i < len(replacements); i++ {
		repl := replacements[i]
		out = preserveAppendString(out, body[repl.start:repl.end])
		out = preserveAppendString(out, repl.text)
	}
	return string(out)
}

func appMain(args []string) int {
	if preserveBody("source-body") != "sourcealphabetaouX" {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}

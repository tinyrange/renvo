package main

type returnedTemp struct {
	name string
	expr string
}

type returnedReplacement struct {
	start int
	end   int
	text  string
}

func buildReturnedTemps() ([]returnedTemp, []returnedReplacement) {
	var temps []returnedTemp
	var replacements []returnedReplacement
	temps = append(temps, returnedTemp{name: "alpha", expr: "beta"})
	temps = append(temps, returnedTemp{name: "gamma", expr: "delta"})
	replacements = append(replacements, returnedReplacement{start: 1, end: 2, text: "x"})
	return temps, replacements
}

func appendReturnedString(out []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	return out
}

func appMain(args []string) int {
	temps, replacements := buildReturnedTemps()
	var out []byte
	for i := 0; i < len(temps); i++ {
		temp := temps[i]
		name := temp.name
		expr := temp.expr
		out = appendReturnedString(out, name)
		out = appendReturnedString(out, ":")
		out = appendReturnedString(out, expr)
		out = append(out, '\n')
	}
	if len(replacements) != 1 || replacements[0].text != "x" {
		print("FAIL\n")
		return 1
	}
	if string(out) != "alpha:beta\ngamma:delta\n" {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}

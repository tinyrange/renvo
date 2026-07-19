package main

type normTemp struct {
	name string
	expr string
}

func appendNormString(out []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	return out
}

func appMain(args []string) int {
	temps := make([]normTemp, 0, 4)
	temps = append(temps, normTemp{name: "alpha", expr: "beta"})
	temps = append(temps, normTemp{name: "gamma", expr: "delta"})
	var out []byte
	for i := 0; i < len(temps); i++ {
		temp := temps[i]
		name := temp.name
		expr := temp.expr
		out = appendNormString(out, name)
		out = appendNormString(out, ":")
		out = appendNormString(out, expr)
		out = append(out, '\n')
	}
	if string(out) != "alpha:beta\ngamma:delta\n" {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}

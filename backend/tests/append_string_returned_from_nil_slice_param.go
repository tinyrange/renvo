package main

func appendString(out []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	return out
}

func makeString(prefix string, value string) string {
	var out []byte
	out = appendString(out, prefix)
	out = appendString(out, value)
	return string(out)
}

func build() string {
	var out []byte
	out = appendString(out, "prefix:")
	out = appendString(out, makeString("alpha", ":one"))
	out = appendString(out, ";")
	out = appendString(out, makeString("beta", ":two"))
	out = appendString(out, ";suffix")
	return string(out)
}

func appMain(args []string, env []string) int {
	got := build()
	if got != "prefix:alpha:one;beta:two;suffix" {
		print(got)
		print("\n")
		return 0
	}
	print("PASS\n")
	return 0
}

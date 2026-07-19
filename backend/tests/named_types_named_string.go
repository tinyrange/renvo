package main

type Renvo0655Name string

func appMain(args []string) int {
	name := renvo0655Make("oak")
	if name != "oak" {
		print("RENVO-0655 named string failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

func renvo0655Make(s string) Renvo0655Name {
	return Renvo0655Name(s)
}

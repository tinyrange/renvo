package main

type Renvo0605Name struct{ name string }

func appMain(args []string) int {
	item := renvo0605Make("elm")
	if item.name != "elm" {
		print("RENVO-0605 string field failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

func renvo0605Make(s string) Renvo0605Name {
	return Renvo0605Name{name: s}
}

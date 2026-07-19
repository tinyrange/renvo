package main

type Renvo0613Flag struct{ ok bool }

func renvo0613All(n int) bool {
	if n == 0 {
		return true
	}
	return renvo0613All(n - 1)
}

func appMain(args []string) int {
	f := Renvo0613Flag{ok: renvo0613All(2)}
	if !f.ok {
		print("RENVO-0613 field bool failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

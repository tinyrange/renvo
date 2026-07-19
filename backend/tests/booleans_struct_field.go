package main

type bool19Box struct{ ok bool }

func appMain(args []string) int {
	x := bool19Box{ok: true}
	if !x.ok {
		print("booleans_19 struct\n")
		return 1
	}
	print("PASS\n")
	return 0
}

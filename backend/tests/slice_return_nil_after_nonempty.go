package main

func names(skip bool) []string {
	if skip {
		return nil
	}
	return []string{"parse"}
}

type Decl struct {
	Names    []string
	Receiver bool
}

func declNames(decl Decl) []string {
	if decl.Receiver {
		return nil
	}
	return decl.Names
}

func appMain(args []string, env []string) int {
	first := names(false)
	if len(first) != 1 {
		print("FAIL\n")
		return 1
	}
	second := names(true)
	if len(second) != 0 {
		print("FAIL\n")
		return 1
	}
	var a Decl
	a.Names = []string{"parse"}
	first = declNames(a)
	if len(first) != 1 {
		print("FAIL\n")
		return 1
	}
	var b Decl
	b.Receiver = true
	second = declNames(b)
	if len(second) != 0 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}

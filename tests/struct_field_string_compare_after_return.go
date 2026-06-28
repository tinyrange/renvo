package main

type Info struct {
	Name string
}

func parseName(name string) Info {
	return Info{Name: name}
}

func appMain(args []string, env []string) int {
	pkgName := ""
	first := parseName("load")
	if len(pkgName) == 0 {
		pkgName = first.Name
	}
	second := parseName("load")
	if pkgName != second.Name {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}

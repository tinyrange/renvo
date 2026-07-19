package main

type Module struct {
	Root string
	Path string
}

func newModule(root string, path string) Module {
	var module Module
	module.Root = root
	module.Path = path
	return module
}

func appMain(args []string, env []string) int {
	m := newModule("root", "path")
	if m.Root != "root" {
		print("FAIL root\n")
		return 1
	}
	if m.Path != "path" {
		print("FAIL path\n")
		return 1
	}
	print("PASS\n")
	return 0
}

package main

type reader interface {
	Read(path string) ([]byte, bool)
}

type fs struct{}

func (f fs) Read(path string) ([]byte, bool) {
	if path != "go.mod" {
		return nil, false
	}
	return []byte("module x\n"), true
}

func use(r reader, path string) bool {
	data, ok := r.Read(path)
	return ok && len(data) == 9
}

func appMain(args []string, env []string) int {
	if !use(fs{}, "go.mod") {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}

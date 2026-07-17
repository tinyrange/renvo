package main

type holder struct {
	value int
}

var unusedPackageVariable int

func unusedParameter(value int) {}

func main() {
	item := holder{}
	item.value = 1
	count := 0
	count += item.value
	blank := count
	_ = blank
	captured := count
	read := func() int { return captured }
	deferred := count
	defer func() { _ = deferred }()
	if read() != 1 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}

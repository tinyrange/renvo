package main

type result struct {
	ok bool
}

type handler func() result

type control struct {
	callback handler
}

type form struct{}

func (f *form) invoke() result {
	return result{ok: true}
}

func main() {
	var f form
	var c control
	c.callback = f.invoke
	if c.callback().ok {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

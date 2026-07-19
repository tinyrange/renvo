package main

type item struct {
	value int
}

type completion func(source []byte, caret int) []item

type editor struct {
	Complete completion
}

type provider struct{}

func (p *provider) complete(source []byte, caret int) []item {
	return []item{{value: len(source) + caret}}
}

func main() {
	var ed editor
	var provider provider
	ed.Complete = provider.complete
	items := ed.Complete([]byte("abc"), 2)
	if len(items) == 1 && items[0].value == 5 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

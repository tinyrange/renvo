package callback

type Handler func()

type item struct {
	activate Handler
}

func Run() {
	var items []*item
	items = append(items, &item{})
	index := 0
	if items[index].activate != nil {
		items[index].activate()
	}
}

package main

type Item struct {
	A string
	B string
	C []string
	D []string
	E []string
	F []string
}

type Module struct {
	Root     string
	Path     string
	Requires []string
	Replaces []string
}

type Graph struct {
	Module Module
	Items  []Item
}

func makeItem() Item {
	var item Item
	item.A = "a"
	item.B = "b"
	return item
}

func addItem(g *Graph, item Item) {
	g.Items = append(g.Items, item)
}

func appMain(args []string, env []string) int {
	var module Module
	module.Path = "module"
	var graph Graph
	graph.Module = module
	g := &graph
	item := makeItem()
	addItem(g, item)
	if len(g.Items) != 1 {
		print("FAIL len\n")
		return 1
	}
	if g.Items[0].A != "a" {
		print("FAIL field\n")
		return 1
	}
	print("PASS\n")
	return 0
}

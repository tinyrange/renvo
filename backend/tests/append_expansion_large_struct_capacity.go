package main

type renvoAppendExpansionRect struct {
	minX float64
	minY float64
	maxX float64
	maxY float64
}

type renvoAppendExpansionNode struct {
	id          string
	parent      string
	name        string
	description string
	value       string
	bounds      renvoAppendExpansionRect
	actions     int
	hidden      bool
	disabled    bool
}

type renvoAppendExpansionControl struct {
	id       string
	children func() []renvoAppendExpansionNode
}

func (c *renvoAppendExpansionControl) makeChildren() []renvoAppendExpansionNode {
	return []renvoAppendExpansionNode{
		{id: c.id + "-one", name: "One"},
		{id: c.id + "-two", name: "Two"},
		{id: c.id + "-three", name: "Three"},
	}
}

func (c *renvoAppendExpansionControl) nodes() []renvoAppendExpansionNode {
	base := renvoAppendExpansionNode{id: c.id}
	children := c.children()
	for i := 0; i < len(children); i++ {
		children[i].parent = base.id
	}
	nodes := []renvoAppendExpansionNode{base}
	return append(nodes, children...)
}

func renvoAppendExpansionSnapshot(controls []*renvoAppendExpansionControl) []renvoAppendExpansionNode {
	nodes := make([]renvoAppendExpansionNode, 0, len(controls))
	for i := 0; i < len(controls); i++ {
		nodes = append(nodes, controls[i].nodes()...)
	}
	return nodes
}

func appMain(args []string) int {
	control := &renvoAppendExpansionControl{id: "probe"}
	control.children = control.makeChildren
	nodes := renvoAppendExpansionSnapshot([]*renvoAppendExpansionControl{control})
	if len(nodes) != 4 || nodes[0].id != "probe" || nodes[1].id != "probe-one" || nodes[1].parent != "probe" || nodes[2].id != "probe-two" || nodes[3].id != "probe-three" {
		return 1
	}
	print("PASS\n")
	return 0
}

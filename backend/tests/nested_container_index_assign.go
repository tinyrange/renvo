package main

type nestedItem struct {
	value int
}

type nestedGroup struct {
	items  []nestedItem
	labels []string
	values []int
}

type nestedGraph struct {
	groups []nestedGroup
}

func appMain() int {
	graph := nestedGraph{groups: []nestedGroup{{
		items:  []nestedItem{{value: 1}},
		labels: []string{"old"},
		values: []int{2},
	}}}
	graph.groups[0].items[0] = nestedItem{value: 7}
	graph.groups[0].labels[0] = "new"
	graph.groups[0].values[0] = 9
	graph.groups[0].values[0] += 2
	if graph.groups[0].items[0].value == 7 && graph.groups[0].labels[0] == "new" && graph.groups[0].values[0] == 11 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}

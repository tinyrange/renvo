package main

type appendExpansionSource struct {
	values []int
}

type appendExpansionDestination struct {
	marker int
	values []int
}

func appMain(args []string) int {
	var sources []appendExpansionSource
	sources = append(sources, appendExpansionSource{values: []int{11}})
	sources = append(sources, appendExpansionSource{values: []int{22}})
	sources = append(sources, appendExpansionSource{values: []int{33}})

	var destinations []appendExpansionDestination
	for i := 0; i < len(sources); i++ {
		destinations = append(destinations, appendExpansionDestination{marker: i})
		index := len(destinations) - 1
		destinations[index].values = append(destinations[index].values, sources[i].values...)
	}

	if len(destinations) != 3 || len(destinations[0].values) != 1 || len(destinations[1].values) != 1 || len(destinations[2].values) != 1 {
		print("FAIL\n")
		return 1
	}
	if destinations[0].values[0] != 11 || destinations[1].values[0] != 22 || destinations[2].values[0] != 33 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}

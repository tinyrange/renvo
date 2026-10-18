package runtime

// Reverse postorder keeps the admitted root first and lays out acyclic paths
// ahead of their sources. Hints only affect layout: all actual transfers still
// compare full guest PCs and apply their original progress/budget checks.
func wasmGraphOrder(blocks []GraphBlock) []GraphBlock {
	visited := make([]bool, len(blocks))
	nodes, edges := []int{0}, []int{0}
	visited[0] = true
	post := make([]int, 0, len(blocks))
	for len(nodes) > 0 {
		top := len(nodes) - 1
		node, edge := nodes[top], edges[top]
		if edge >= blocks[node].SuccessorCount {
			post = append(post, node)
			nodes, edges = nodes[:top], edges[:top]
			continue
		}
		edges[top]++
		target := blocks[node].Successors[edge]
		for next := 0; next < len(blocks); next++ {
			if blocks[next].PC == target {
				if !visited[next] {
					visited[next] = true
					nodes = append(nodes, next)
					edges = append(edges, 0)
				}
				break
			}
		}
	}
	ordered := make([]GraphBlock, 0, len(blocks))
	for i := len(post) - 1; i >= 0; i-- {
		ordered = append(ordered, blocks[post[i]])
	}
	for i, block := range blocks {
		if !visited[i] {
			ordered = append(ordered, block)
		}
	}
	return ordered
}

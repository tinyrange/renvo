package check

type genericPromotionNode struct {
	typ     int
	pointer bool
	path    []int
}

// Search one embedding depth at a time. Both fields and methods shadow names
// at greater depths, and two paths to a name at the same depth are ambiguous.
func (t *genericTypes) promotedMethods(id int) []genericMethod {
	return t.promotedMembers(id, false)
}

func (t *genericTypes) promotedMembers(id int, fields bool) []genericMethod {
	var out, claimed []genericMethod
	pending := []genericPromotionNode{{typ: id}}
	for len(pending) > 0 {
		var next []genericPromotionNode
		var names []genericMethod
		for _, node := range pending {
			v := t.get(node.typ)
			if v.kind == genericPointer {
				node.typ, node.pointer = v.elem, true
				if t.get(t.underlying(node.typ)).kind == genericInterface {
					continue
				}
			}
			cycle := false
			for _, old := range node.path {
				if old == node.typ {
					cycle = true
				}
			}
			if cycle {
				continue
			}
			path := append(append([]int(nil), node.path...), node.typ)
			for _, method := range t.directMethods(node.typ, node.pointer) {
				if fields {
					method.typ = 0
				}
				names = append(names, method)
			}
			u := t.get(t.underlying(node.typ))
			if u.kind != genericStruct {
				continue
			}
			for _, field := range u.fields {
				typ := 0
				if fields {
					typ = field.typ
				}
				names = append(names, genericMethod{name: field.name, pkg: field.pkg, typ: typ, embedded: field.embedded})
				if field.embedded {
					next = append(next, genericPromotionNode{typ: field.typ, pointer: node.pointer, path: path})
				}
			}
		}
		for i, name := range names {
			count := 0
			for _, old := range claimed {
				if genericSameMethodName(name, old) {
					count++
				}
			}
			if count > 0 {
				continue
			}
			for _, other := range names {
				if genericSameMethodName(name, other) {
					count++
				}
			}
			if count == 1 && name.typ != 0 {
				out = append(out, name)
			}
			claimed = append(claimed, names[i])
		}
		pending = next
	}
	return out
}

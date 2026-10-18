package rtg

// Compiler ABI parameters are static name/type pairs. A shared constructor
// avoids emitting hundreds of independent slice/struct initializers into the
// self-hosted compiler. The table uses "name type,name type" spelling; ABI types
// here are identifiers, pointers and slices, without whitespace or commas.
func compilerParameters(spec string) []compilerBindingParameter {
	count := 0
	if len(spec) != 0 {
		count = 1
		for i := 0; i < len(spec); i++ {
			if spec[i] == ',' {
				count++
			}
		}
	}
	out := make([]compilerBindingParameter, count)
	start := 0
	for i := 0; i < count; i++ {
		split := start
		for split < len(spec) && spec[split] != ' ' && spec[split] != ',' {
			split++
		}
		end := split + 1
		for end < len(spec) && spec[end] != ',' {
			end++
		}
		if split == start || split >= len(spec) || spec[split] != ' ' || end == split+1 {
			panic("invalid compiler parameter specification")
		}
		out[i] = compilerBindingParameter{Name: spec[start:split], Type: spec[split+1 : end]}
		start = end + 1
	}
	return out
}

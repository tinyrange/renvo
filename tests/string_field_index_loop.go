package main

type Holder struct {
	value string
}

func countLeadingSpaces(holder Holder) int {
	i := 0
	for i < len(holder.value) {
		c := holder.value[i]
		if c != ' ' && c != '\t' && c != '\n' {
			break
		}
		i++
	}
	return i
}

func appMain(args []string, env []string) int {
	holder := Holder{value: "\n\t  text"}
	if countLeadingSpaces(holder) != 4 {
		print("bad count\n")
		return 1
	}
	print("PASS\n")
	return 0
}

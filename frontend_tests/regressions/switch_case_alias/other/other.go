package other

const Set = "set"
const Last = "last"
const Get = "get"

type Item struct{ Set int }

func choose(s string) int {
	switch s {
	case Get, Set:
		return 3
	case Last:
		return 4
	}
	return 0
}

func Run() {
	item := Item{Set: 7}
Set:
	for item.Set > 0 {
		break Set
	}
	if choose("set") != 3 || choose("get") != 3 || choose("last") != 4 || choose("absent") != 0 || item.Set != 7 {
		panic("case alias or field/label identity")
	}
	println("PASS")
}

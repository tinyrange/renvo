package fronttrace

var enabled bool

func SetEnabled(value bool) { enabled = value }

func Event(value string) {
	if !enabled {
		return
	}
	print("renvo frontend: ")
	print(value)
	print("\n")
}

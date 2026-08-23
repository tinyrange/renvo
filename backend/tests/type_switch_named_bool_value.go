package main

type typeSwitchBoolValue interface {
	Type() string
}

type typeSwitchBool bool

func (typeSwitchBool) Type() string { return "bool" }

func typeSwitchTruth(value typeSwitchBoolValue) bool {
	{
		dirty := uint64(0x0100)
		if dirty == 0 {
			return false
		}
	}
	switch x := value.(type) {
	case typeSwitchBool:
		return bool(x)
	}
	return true
}

func appMain() int {
	if !typeSwitchTruth(typeSwitchBool(false)) {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}

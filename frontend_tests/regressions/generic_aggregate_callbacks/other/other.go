package other

type Private = struct{ value int }

func PrivateValue(v Private) int { return v.value }

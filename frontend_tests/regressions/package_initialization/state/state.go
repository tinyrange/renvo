package state

func source() int { return 9 }

var Trace = source()

func init() { Trace = Trace*10 + 1 }
func init() { Trace = Trace*10 + 2 }

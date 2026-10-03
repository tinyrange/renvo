package callback

func Unique() int     { return 17 }
func Id[T any](v T) T { return v }

type Type struct{ N int }

var Counter = 19

const Width = 3

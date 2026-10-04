package callback

func First() int      { return 17 }
func Second() int     { return 19 }
func Id[T any](v T) T { return v }

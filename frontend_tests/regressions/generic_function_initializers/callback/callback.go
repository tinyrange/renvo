package callback

func First() int      { return 17 }
func Second() int     { return 19 }
func Id[T any](v T) T { return v }

type Fn func() int

func (f Fn) Read() int { return f() }

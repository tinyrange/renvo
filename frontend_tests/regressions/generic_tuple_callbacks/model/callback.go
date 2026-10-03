package model

func Identity[T any](v T) T { return v }
func Pair() (int, any)      { return 5, nil }
func Boxed() (int, any)     { return 7, "value" }

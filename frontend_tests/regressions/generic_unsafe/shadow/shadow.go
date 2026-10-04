package shadow

const true = 7

var false = 8

func nil() int { return 9 }

const iota = 10

func Sum[T ~int]() T { return T(true + false + nil() + iota) }

package callback

type Base struct{ Value int }
type base int

var Initialized int

func init()           { Initialized = 5 }
func Box() any        { return Base{Value: 42} }
func Value() int      { return 17 }
func PrivateBox() any { return base(43) }
func Id[T any](v T) T { return v }

package callback

var Counter = 17
var Initialized int

func init()                      { Initialized = 5 }
func Unique() int                { return 17 }
func Id[T any](v T) T            { return v }
func Apply[T any](fn func() T) T { return fn() }

func Other() int { return 19 }

type Type struct{ N int }

func (v Type) init() int { return v.N }
func ReadType() int      { return Type{42}.init() }

type Box[T any] struct{ Value T }

func (b Box[T]) init() T { return b.Value }
func Read[T any](v T) T  { return Box[T]{v}.init() }

package sequence

type hidden struct{ N int }

func Items[V ~int](v V) func(func(hidden) bool) {
	return func(yield func(hidden) bool) { yield(hidden{N: int(v)}) }
}

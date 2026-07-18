package lib

type Pair struct {
	Left  int
	Right int
}

type Scorer interface {
	Score(int) int
	Pair() Pair
	Split() (int, int)
}

type Narrow interface {
	Score(int) int
}

type Alpha struct {
	Base int
}

func (value Alpha) Score(extra int) int {
	return value.Base + extra
}

func (value Alpha) Pair() Pair {
	return Pair{Left: value.Base, Right: value.Base + 1}
}

func (value Alpha) Split() (int, int) {
	return value.Base, value.Base + 2
}

type Beta struct {
	Base int
}

func (value *Beta) Score(extra int) int {
	return value.Base*2 + extra
}

func (value *Beta) Pair() Pair {
	return Pair{Left: value.Base * 2, Right: value.Base * 3}
}

func (value *Beta) Split() (int, int) {
	return value.Base * 4, value.Base * 5
}

type Unrelated struct{}

func (Unrelated) Score(extra int) string {
	return "unrelated"
}

func Choose(alpha bool) Scorer {
	if alpha {
		return Alpha{Base: 10}
	}
	return &Beta{Base: 20}
}

func NarrowFrom(value Scorer) Narrow {
	return value
}

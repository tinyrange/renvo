package main

type T struct{}

type TestCase struct {
	Run func(*T)
}

func (t *T) Run(name string, callback func(value *T)) {
	callback(t)
}

func exercise(t *T) {
	for name, source := range map[string]string{"case": "value"} {
		t.Run("captured", func(t *T) {
			if name != "case" || source != "value" {
				panic("capture")
			}
		})
	}
	t.Run("named", check)
}

func check(t *T) {
	if t == nil {
		panic("missing test")
	}
}

func main() {
	test := TestCase{Run: exercise}
	test.Run(&T{})
	print("PASS\n")
}

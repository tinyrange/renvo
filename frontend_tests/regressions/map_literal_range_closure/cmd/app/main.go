package main

type Receiver struct{ calls int }

func (r *Receiver) Run(name string, f func(*Receiver)) { r.calls++; f(r) }

func main() {
	r := &Receiver{}
	total := 0
	for name, value := range map[string]string{"one": "a", "two": "bb"} {
		r.Run(name, func(r *Receiver) { total += len(value) })
	}
	if total != 3 || r.calls != 2 {
		panic("map literal range lost key/value semantics")
	}
	print("PASS\n")
}

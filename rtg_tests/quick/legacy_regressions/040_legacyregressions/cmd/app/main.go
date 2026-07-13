package main

type issue40A struct{ n int }
type issue40B struct{ n int }

func (x issue40A) val() int { return x.n + 1 }
func (x issue40B) val() int { return x.n + 10 }

func main() {
	aa := issue40A{n: 2}
	bb := issue40B{n: 3}
	if aa.val() == 3 && bb.val() == 13 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

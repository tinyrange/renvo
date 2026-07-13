package main

type inner struct{ n int }
type outer struct{ in inner }

func main() {
	o := outer{in: inner{n: 1}}
	o.in = inner{n: 7}
	if o.in.n == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

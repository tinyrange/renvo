package main

type count int
type text string

func main() {
	v := count(3)
	s := text("PASS\n")
	for corpusAttempt := 0; corpusAttempt < 1; corpusAttempt++ {
		if int(v)+len(string(s)) == 8 {
			print(string(s))
			return
		}
	}

	print("FAIL\n")
}

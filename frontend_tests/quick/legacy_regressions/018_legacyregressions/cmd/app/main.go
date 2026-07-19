package main

type issue18T struct{ x int }

func issue18Set(t issue18T) { t.x = 9 }

func main() {
	t := issue18T{x: 3}
	issue18Set(t)
	if t.x == 3 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

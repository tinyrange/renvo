package main

func issue24Pair() (int, string) { return 3, "ok" }

func main() {
	_, s := issue24Pair()
	if s == "ok" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

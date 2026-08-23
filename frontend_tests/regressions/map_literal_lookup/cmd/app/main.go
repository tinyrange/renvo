package main

func main() {
	key := 7
	value := map[int]string{key: "PASS\n"}[key]
	if value != "PASS\n" {
		print("FAIL\n")
		return
	}
	print(value)
}

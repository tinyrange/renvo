package main

func main() {
	if platformValue()+legacyValue()+modernValue() == 44 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

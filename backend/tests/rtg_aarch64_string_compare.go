package main

func appMain(args []string) int {
	data := []byte{'x', 'O', 'S', '/', '2', 'y'}
	if string(data[1:5]) != "OS/2" {
		print("FAIL\n")
		return 1
	}
	if string(data[1:5]) == "cmap" {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}

package main

type rtgNamedString string

func appMain(args []string) int {
	value := rtgNamedString("PASS\n")
	if len(string(value)) != 5 {
		print("RTG named string conversion failed\n")
		return 1
	}
	print(string(value))
	return 0
}

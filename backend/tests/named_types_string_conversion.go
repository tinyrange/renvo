package main

type renvoNamedString string

func appMain(args []string) int {
	value := renvoNamedString("PASS\n")
	if len(string(value)) != 5 {
		print("RENVO named string conversion failed\n")
		return 1
	}
	print(string(value))
	return 0
}

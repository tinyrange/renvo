package main

type Renvo0659Count int

func appMain(args []string) int {
	var n Renvo0659Count = 11
	total := 0
	for i := 0; i < int(n); i = i + 1 {
		total = total + 1
	}
	if total != 11 {
		print("RENVO-0659 named int conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

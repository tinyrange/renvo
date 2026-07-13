package main

func appMain() int {
	first := make(map[string]int)
	second := make(map[string]int)
	for i := 0; i < 2; i++ {
		current := make(map[string]int, 1)
		current["same"] = i + 1
		if i == 0 {
			first = current
		} else {
			second = current
		}
	}
	if first["same"] != 1 || second["same"] != 2 {
		print("FAIL\n")
		return 1
	}

	grown := make(map[string]int, 1)
	grown["k00"] = 0
	grown["k01"] = 1
	grown["k02"] = 2
	grown["k03"] = 3
	grown["k04"] = 4
	grown["k05"] = 5
	grown["k06"] = 6
	grown["k07"] = 7
	grown["k08"] = 8
	grown["k09"] = 9
	grown["k10"] = 10
	grown["k11"] = 11
	grown["k12"] = 12
	grown["k13"] = 13
	grown["k14"] = 14
	grown["k15"] = 15
	grown["k16"] = 16
	grown["k17"] = 17
	grown["k18"] = 18
	grown["k19"] = 19
	for key, value := range grown {
		if key == "k19" && value != 19 {
			print("FAIL\n")
			return 2
		}
	}
	if grown["k00"]+grown["k07"]+grown["k16"]+grown["k19"] != 42 {
		print("FAIL\n")
		return 3
	}
	print("PASS\n")
	return 0
}

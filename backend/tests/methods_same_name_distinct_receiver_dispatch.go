package main

type renvoLegacy40A struct{ n int }
type renvoLegacy40B struct{ n int }

func (x renvoLegacy40A) Value() int { return x.n + 1 }
func (x renvoLegacy40B) Value() int { return x.n + 10 }

func appMain(args []string) int {
	a := renvoLegacy40A{n: 2}
	b := renvoLegacy40B{n: 3}
	if a.Value() != 3 || b.Value() != 13 {
		print("methods_same_name_distinct_receiver_dispatch failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

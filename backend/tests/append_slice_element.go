package main

func appMain(args []string) int {
	var rows [][]string
	row := []string{"value"}
	rows = append(rows, row)
	if len(rows) != 1 || len(rows[0]) != 1 || rows[0][0] != "value" {
		return 1
	}
	print("PASS\n")
	return 0
}

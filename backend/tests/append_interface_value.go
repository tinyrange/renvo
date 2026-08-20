package main

func appMain() int {
	values := make([]any, 0)
	var value any = "renvo"
	values = append(values, value)
	values = append(values, 7)
	text, textOK := values[0].(string)
	number, numberOK := values[1].(int)
	if len(values) == 2 && textOK && text == "renvo" && numberOK && number == 7 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}

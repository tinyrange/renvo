package main

func aliasCallback(v string) string { return v + "!" }

func appMain() int {
	{
		type int = string
		if int("value") != "value" {
			return 1
		}
	}
	{
		type string = int
		if string(42) != 42 {
			return 2
		}
	}
	{
		type bool = int
		if bool(42) != 42 {
			return 3
		}
	}
	{
		type float64 = string
		if float64("float") != "float" {
			return 4
		}
	}
	{
		type complex64 = string
		if complex64("complex") != "complex" {
			return 5
		}
	}
	{
		type uint8 = string
		if uint8("byte") != "byte" {
			return 6
		}
	}
	{
		type Callback = func(int) int
		f := Callback(func(v int) int { return v + 1 })
		if f(41) != 42 {
			return 13
		}
	}
	int := func(v string) string { return v + "!" }
	{
		type int = string
		if int("inner") != "inner" {
			return 7
		}
		{
			int := func(v string) string { return v + "?" }
			if int("value") != "value?" {
				return 8
			}
		}
		if int("outer") != "outer" {
			return 9
		}
	}
	if int("callback") != "callback!" {
		return 10
	}
	{
		type aliasCallback = string
		if aliasCallback("hidden") != "hidden" {
			return 11
		}
	}
	if aliasCallback("call") != "call!" {
		return 12
	}
	print("PASS\n")
	return 0
}

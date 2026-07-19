package main

type renvo_example_com_regression_cmd_app_Box struct {
	Inner renvo_example_com_regression_cmd_app_Inner
}

type renvo_example_com_regression_cmd_app_Inner struct {
	Tag string
}

func renvo_example_com_regression_cmd_app_same(left renvo_example_com_regression_cmd_app_Box, right renvo_example_com_regression_cmd_app_Box) bool {
	return left.Inner.Tag == right.Inner.Tag
}

func appMain(args []string, env []string) int {
	left := renvo_example_com_regression_cmd_app_Box{Inner: renvo_example_com_regression_cmd_app_Inner{Tag: "b"}}
	right := renvo_example_com_regression_cmd_app_Box{Inner: renvo_example_com_regression_cmd_app_Inner{Tag: "b"}}
	if renvo_example_com_regression_cmd_app_same(left, right) {
		if left.Inner.Tag == right.Inner.Tag {
			print("PASS\n")
			return 0
		}
	}
	print("FAIL\n")
	return 0
}

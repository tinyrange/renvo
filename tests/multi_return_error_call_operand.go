package main

type config struct {
	value int
}

type testError string

func (err testError) Error() string {
	return string(err)
}

func makeError() error {
	return testError("bad")
}

func parse() (config, error) {
	cfg := config{value: 9}
	return cfg, makeError()
}

func appMain() int {
	cfg, err := parse()
	if cfg.value != 9 {
		print("multi_return_error_call_operand config failed\n")
		return 1
	}
	if err == nil {
		print("multi_return_error_call_operand error failed\n")
		return 1
	}
	if err.Error() != "bad" {
		print("multi_return_error_call_operand message failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

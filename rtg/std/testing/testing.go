package testing

type T struct {
	name   string
	failed bool
	logs   []string
}

type InternalTest struct {
	Name string
	F    func(*T)
}

type InternalBenchmark struct{}
type InternalExample struct{}

type failNow struct{}

func (t *T) Fail() {
	t.failed = true
}

func (t *T) FailNow() {
	t.Fail()
	panic(failNow{})
}

func (t *T) Failed() bool {
	return t.failed
}

func (t *T) Error(args ...interface{}) {
	t.Log(args...)
	t.Fail()
}

func (t *T) Errorf(format string, args ...interface{}) {
	t.Logf(format, args...)
	t.Fail()
}

func (t *T) Fatal(args ...interface{}) {
	t.Log(args...)
	t.FailNow()
}

func (t *T) Fatalf(format string, args ...interface{}) {
	t.Logf(format, args...)
	t.FailNow()
}

func (t *T) Log(args ...interface{}) {
	t.logs = append(t.logs, sprint(args...))
}

func (t *T) Logf(format string, args ...interface{}) {
	t.logs = append(t.logs, sprintf(format, args...))
}

func (t *T) Helper() {}

func (t *T) Run(name string, f func(t *T)) bool {
	child := &T{name: t.name + "/" + name}
	f(child)
	if child.failed {
		t.Fail()
	}
	return !child.failed
}

func Main(matchString func(pat string, str string) (bool, error), tests []InternalTest, benchmarks []InternalBenchmark, examples []InternalExample) {
	failed := false
	for i := 0; i < len(tests); i++ {
		test := tests[i]
		run := true
		if matchString != nil {
			ok, err := matchString("", test.Name)
			run = err == nil && ok
		}
		if !run {
			continue
		}
		t := &T{name: test.Name}
		func() {
			defer func() {
				if recover() != nil {
					t.Fail()
				}
			}()
			test.F(t)
		}()
		if t.failed {
			failed = true
			print("--- FAIL: ")
			print(test.Name)
			print("\n")
		}
	}
	if failed {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}

func sprint(args ...interface{}) string {
	out := ""
	for i := 0; i < len(args); i++ {
		if i > 0 {
			out += " "
		}
		out += valueString(args[i])
	}
	return out
}

func sprintf(format string, args ...interface{}) string {
	out := ""
	arg := 0
	for i := 0; i < len(format); i++ {
		if format[i] != '%' || i+1 >= len(format) {
			out += string(format[i])
			continue
		}
		i++
		if format[i] == '%' {
			out += "%"
			continue
		}
		if arg >= len(args) {
			out += "%!"
			out += string(format[i])
			continue
		}
		out += valueString(args[arg])
		arg++
	}
	return out
}

func valueString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case int:
		return intString(x)
	case bool:
		if x {
			return "true"
		}
		return "false"
	case error:
		return x.Error()
	}
	return "<?>"
}

func intString(v int) string {
	if v < 0 {
		return "-" + uintString(uint(-v))
	}
	return uintString(uint(v))
}

func uintString(v uint) string {
	if v == 0 {
		return "0"
	}
	var buf [32]byte
	pos := len(buf)
	for v > 0 {
		pos--
		buf[pos] = byte('0' + v%10)
		v = v / 10
	}
	return string(buf[pos:])
}

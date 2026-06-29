package fmt

type fmtError string

func (err fmtError) Error() string {
	return string(err)
}

type Error string

func (err Error) Error() string {
	return string(err)
}

func Print(s string) int {
	print(s)
	return 0
}

func Println(s string) int {
	print(s)
	print("\n")
	return 0
}

func PrintString(s string) int {
	print(s)
	return 0
}

func PrintInt(v int) int {
	print(v)
	return 0
}

func Fprint(fd int, s string) int {
	return Print(s)
}

func Fprintln(fd int, s string) int {
	Print(s)
	print("\n")
	return 0
}

func Fprintf(fd int, format string) int {
	print(format)
	return 0
}

func Errorf(format string) error {
	return Error(format)
}

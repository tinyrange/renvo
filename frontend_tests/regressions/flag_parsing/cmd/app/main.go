package main

import (
	"bytes"
	"flag"
	"fmt"
)

func main() {
	f := flag.NewFlagSet("test", flag.ContinueOnError)
	var output bytes.Buffer
	f.SetOutput(&output)
	n := f.Uint64("gas", 100, "budget")
	text := f.String("text", "default", "text")
	enabled := f.Bool("verbose", false, "verbose")
	if err := f.Parse([]string{"--gas=0xffffffffffffffff", "-text", "hello", "-verbose", "input", "-tail"}); err != nil {
		panic(err)
	}
	if *n != 18446744073709551615 || *text != "hello" || !*enabled || f.NFlag() != 3 || f.NArg() != 2 || f.Arg(0) != "input" {
		panic("flags")
	}
	if err := f.Set("gas", "9"); err != nil || *n != 9 || f.NFlag() != 3 {
		panic("set")
	}
	getter, ok := f.Lookup("gas").Value.(flag.Getter)
	if !ok || getter.Get() != uint64(9) {
		panic("getter")
	}
	if err := f.Parse([]string{"-unknown"}); err == nil || output.Len() == 0 {
		panic("diagnostic")
	}
	fmt.Println("PASS")
}

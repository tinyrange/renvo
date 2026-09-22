package flag

import (
	"bytes"
	"strings"
	"testing"
)

func TestFlagParsing(t *testing.T) {
	f := NewFlagSet("test", ContinueOnError)
	n := f.Uint64("gas", 100, "budget")
	text := f.String("text", "default", "text")
	b := f.Bool("verbose", false, "verbose")
	signed := f.Int("signed", 0, "signed")
	if err := f.Parse([]string{"--gas=0xffffffffffffffff", "-text", "hello", "-verbose", "-signed", "-42", "input.star", "-other"}); err != nil {
		t.Fatal(err)
	}
	if *n != 18446744073709551615 || *text != "hello" || !*b || *signed != -42 {
		t.Fatal("typed parsing")
	}
	if !f.Parsed() || f.NArg() != 2 || f.Arg(0) != "input.star" || f.Arg(1) != "-other" || f.Arg(-1) != "" || f.Arg(2) != "" {
		t.Fatal("positional args")
	}
	if f.NFlag() != 4 || f.Name() != "test" || f.ErrorHandling() != ContinueOnError {
		t.Fatal("flag set state")
	}
	var names string
	f.Visit(func(item *Flag) { names += item.Name + "," })
	if names != "gas,signed,text,verbose," {
		t.Fatal("visit order")
	}
	if err := f.Set("gas", "9"); err != nil || *n != 9 || f.NFlag() != 4 {
		t.Fatal("set")
	}
	getter, ok := f.Lookup("gas").Value.(Getter)
	if !ok || getter.Get() != uint64(9) {
		t.Fatal("getter")
	}
}

func TestFlagErrorsAndTermination(t *testing.T) {
	for _, args := range [][]string{{"-missing"}, {"---bad"}, {"-gas"}, {"-gas=18446744073709551616"}, {"-gas=-1"}, {"-verbose=maybe"}} {
		f := NewFlagSet("test", ContinueOnError)
		f.Uint64("gas", 100, "budget")
		f.Bool("verbose", false, "verbose")
		var output bytes.Buffer
		f.SetOutput(&output)
		usages := 0
		f.Usage = func() { usages++ }
		if err := f.Parse(args); err == nil || usages != 1 || output.Len() == 0 {
			t.Fatal("missing parse diagnostic")
		}
	}
	f := NewFlagSet("test", ContinueOnError)
	f.Bool("verbose", false, "verbose")
	if err := f.Parse([]string{"--", "-verbose"}); err != nil || f.NFlag() != 0 || f.Arg(0) != "-verbose" {
		t.Fatal("terminator")
	}
	var output bytes.Buffer
	f.SetOutput(&output)
	if err := f.Parse([]string{"-h"}); err != ErrHelp || !strings.Contains(output.String(), "Usage of test:") {
		t.Fatal("help")
	}
	f.Bool("h", false, "explicit help flag")
	if err := f.Parse([]string{"-h"}); err != nil {
		t.Fatal("explicit h", err)
	}
}

func TestFlagDefaults(t *testing.T) {
	f := NewFlagSet("test", ContinueOnError)
	f.Uint64("gas", 100, "execution `budget`")
	f.String("mode", "false", "mode")
	var output bytes.Buffer
	f.SetOutput(&output)
	f.PrintDefaults()
	if !strings.Contains(output.String(), "-gas budget") || !strings.Contains(output.String(), "(default 100)") || !strings.Contains(output.String(), "(default \"false\")") {
		t.Fatal("defaults", output.String())
	}
	name, usage := UnquoteUsage(f.Lookup("gas"))
	if name != "budget" || usage != "execution budget" {
		t.Fatal("usage placeholder")
	}
}

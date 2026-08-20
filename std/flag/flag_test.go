package flag

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseScalarFlags(t *testing.T) {
	set := NewFlagSet("test", ContinueOnError)
	name := set.String("name", "default", "")
	verbose := set.Bool("verbose", false, "")
	count := set.Int("count", 1, "")
	if err := set.Parse([]string{"--name=renvo", "-verbose", "-count", "3", "arg", "tail"}); err != nil {
		t.Fatal(err)
	}
	if *name != "renvo" || !*verbose || *count != 3 || set.NFlag() != 3 || set.NArg() != 2 || set.Arg(1) != "tail" {
		t.Fatal(*name, *verbose, *count, set.Args())
	}
}
func TestParseErrorsAndTerminator(t *testing.T) {
	set := NewFlagSet("test", ContinueOnError)
	value := set.String("value", "", "")
	if err := set.Parse([]string{"--", "-value=x"}); err != nil || *value != "" || set.Arg(0) != "-value=x" {
		t.Fatal(err, *value, set.Args())
	}
	if NewFlagSet("test", ContinueOnError).Parse([]string{"-missing"}) == nil {
		t.Fatal("missing flag accepted")
	}
}
func TestVisitSorted(t *testing.T) {
	set := NewFlagSet("test", ContinueOnError)
	set.Bool("z", false, "")
	set.Bool("a", false, "")
	set.Parse([]string{"-z", "-a"})
	got := ""
	set.Visit(func(flag *Flag) { got += flag.Name })
	if got != "az" {
		t.Fatal(got)
	}
}

func TestFloatDurationAndDefaults(t *testing.T) {
	set := NewFlagSet("test", ContinueOnError)
	ratio := set.Float64("ratio", 1.5, "set `ratio`")
	delay := set.Duration("delay", 0, "wait `duration`")
	if err := set.Parse([]string{"-ratio=2.5e1", "-delay", "1h2m3.5s"}); err != nil {
		t.Fatal(err)
	}
	if *ratio != 25 || *delay != 3723500000000 {
		t.Fatal(*ratio, *delay)
	}
	var output bytes.Buffer
	set.SetOutput(&output)
	set.PrintDefaults()
	if !strings.Contains(output.String(), "-delay duration") || !strings.Contains(output.String(), "-ratio ratio") {
		t.Fatal(output.String())
	}
}

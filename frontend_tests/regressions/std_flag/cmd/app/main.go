package main

import "flag"

func main() {
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	if set.Usage == nil {
		print("FAIL\n")
		return
	}
	name := set.String("name", "", "")
	verbose := set.Bool("verbose", false, "")
	count := set.Int("count", 0, "")
	err := set.Parse([]string{"--name=renvo", "-verbose", "-count", "3", "arg"})
	if err != nil || *name != "renvo" || !*verbose || *count != 3 || set.Arg(0) != "arg" {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}

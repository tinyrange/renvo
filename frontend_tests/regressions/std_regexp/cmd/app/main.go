package main

import (
	"regexp"
	"strings"
)

func expect(cond bool) {
	if !cond {
		print("FAIL\n")
	}
}

func main() {
	re := regexp.MustCompile("(\\w+)@(\\w+)\\.com")
	if !re.MatchString("mail bob@example.com now") || re.MatchString("no address") {
		expect(false)
		return
	}
	if re.FindString("mail bob@example.com now") != "bob@example.com" {
		expect(false)
		return
	}
	sub := re.FindStringSubmatch("from alice@test.com!")
	if len(sub) != 3 || sub[1] != "alice" || sub[2] != "test" {
		expect(false)
		return
	}
	digits := regexp.MustCompile("\\d+")
	all := digits.FindAllString("a1 b22 c333", -1)
	if len(all) != 3 || all[2] != "333" {
		expect(false)
		return
	}
	swap := regexp.MustCompile("(\\w+) (\\w+)")
	if swap.ReplaceAllString("hello world", "$2 $1") != "world hello" {
		expect(false)
		return
	}
	if !regexp.MatchString("^https?://[^/]+", "https://example.com") {
		expect(false)
		return
	}
	if regexp.QuoteMeta("a.b") != "a\\.b" {
		expect(false)
		return
	}
	lazy := regexp.MustCompile("a+?")
	if lazy.FindString("aaab") != "a" {
		expect(false)
		return
	}
	// Catastrophic-backtracking patterns stay linear-time.
	bomb := regexp.MustCompile("(a+)+b")
	input := strings.Repeat("a", 40)
	if bomb.MatchString(input) {
		expect(false)
		return
	}
	if _, err := regexp.Compile("a(b"); err == nil {
		expect(false)
		return
	}
	print("PASS\n")
}

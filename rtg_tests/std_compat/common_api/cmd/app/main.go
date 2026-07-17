package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
)

type sink struct {
	data []byte
}

func (s *sink) Write(data []byte) (int, error) {
	s.data = append(s.data, data...)
	return len(data), nil
}

func main() {
	parts := bytes.Split([]byte("a,b,c"), []byte(","))
	if len(parts) != 3 || string(bytes.Join(parts, []byte("|"))) != "a|b|c" || !bytes.Equal(bytes.Repeat([]byte("x"), 3), []byte("xxx")) {
		print("FAIL\n")
		return
	}
	buffer := bytes.NewBufferString("go")
	if _, err := buffer.WriteString("pher"); err != nil || buffer.String() != "gopher" || buffer.Len() != 6 {
		print("FAIL\n")
		return
	}
	readBuffer := make([]byte, 8)
	if count, err := buffer.Read(readBuffer); err != nil || count != 6 || string(readBuffer[:count]) != "gopher" {
		print("FAIL\n")
		return
	}
	if count, err := buffer.Read(readBuffer); count != 0 || err == nil || err.Error() != "EOF" {
		print("FAIL\n")
		return
	}
	firstError := errors.New("problem")
	if firstError.Error() != "problem" || !errors.Is(firstError, firstError) || errors.Is(firstError, errors.New("problem")) || !errors.Is(nil, nil) {
		print("FAIL\n")
		return
	}
	if fmt.Sprintf("%s:%d:%x:%q:%t:%%", "id", -7, 255, "go\n", true) != "id:-7:ff:\"go\\n\":true:%" {
		print("FAIL\n")
		return
	}
	if fmt.Sprint("value", 12, true) != "value12 true" {
		print("FAIL\n")
		return
	}
	var output sink
	writeCount, writeErr := fmt.Fprintf(&output, "%s=%d", "value", 12)
	if writeErr != nil {
		print("FAIL\n")
		return
	}
	if writeCount != 8 {
		print("FAIL\n")
		return
	}
	if string(output.data) != "value=12" {
		print("FAIL\n")
		return
	}
	if path.Clean("/a/../b//c") != "/b/c" || path.Join("a", "b", "..", "c") != "a/c" || path.Base("/a/b.txt") != "b.txt" || path.Dir("/a/b.txt") != "/a" || path.Ext("/a/b.txt") != ".txt" {
		print("FAIL\n")
		return
	}
	dir, file := path.Split("/a/b.txt")
	if dir != "/a/" || file != "b.txt" {
		print("FAIL\n")
		return
	}

	ints := []int{4, 1, 3, 2}
	sort.Ints(ints)
	strings := []string{"b", "c", "a"}
	sort.Strings(strings)
	if ints[0] != 1 || ints[3] != 4 || strings[0] != "a" || strings[2] != "c" || sort.Search(10, func(index int) bool { return index >= 6 }) != 6 {
		print("FAIL\n")
		return
	}

	if strconv.Itoa(-42) != "-42" || strconv.FormatInt(255, 16) != "ff" || strconv.FormatUint(35, 36) != "z" || strconv.FormatBool(false) != "false" {
		print("FAIL\n")
		return
	}
	if value, err := strconv.Atoi("-17"); err != nil || value != -17 {
		print("FAIL\n")
		return
	}
	if value, err := strconv.ParseUint("0b101", 0, 64); err != nil || value != 5 {
		print("FAIL\n")
		return
	}
	quoted := strconv.Quote("a\n\"b\"")
	if unquoted, err := strconv.Unquote(quoted); err != nil || unquoted != "a\n\"b\"" {
		print("FAIL\n")
		return
	}

	filename := "/tmp/rtgx-std-common-api.tmp"
	if err := os.WriteFile(filename, []byte("hello"), 0644); err != nil {
		print("FAIL\n")
		return
	}
	if data, err := os.ReadFile(filename); err != nil || string(data) != "hello" {
		print("FAIL\n")
		return
	}
	opened, err := os.Open(filename)
	if err != nil {
		print("FAIL\n")
		return
	}
	fileBuffer := make([]byte, 2)
	if count, readErr := opened.Read(fileBuffer); readErr != nil || count != 2 || string(fileBuffer) != "he" {
		print("FAIL\n")
		return
	}
	if err := opened.Close(); err != nil {
		print("FAIL\n")
		return
	}
	if wd, err := os.Getwd(); err != nil || wd == "" || len(os.Environ()) == 0 {
		print("FAIL\n")
		return
	}

	print("PASS\n")
}

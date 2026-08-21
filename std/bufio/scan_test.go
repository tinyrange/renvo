package bufio

import (
	"bytes"
	"strings"
	"testing"
)

func collect(scanner *Scanner) ([]string, error) {
	var tokens []string
	for scanner.Scan() {
		tokens = append(tokens, scanner.Text())
	}
	return tokens, scanner.Err()
}

func TestScanLinesCRLFFinalLine(t *testing.T) {
	tokens, err := collect(NewScanner(bytes.NewBufferString("data: one\r\n\ndata: two")))
	want := []string{"data: one", "", "data: two"}
	if err != nil || len(tokens) != len(want) {
		t.Fatalf("tokens=%q err=%v", tokens, err)
	}
	for i := range want {
		if tokens[i] != want[i] {
			t.Fatalf("tokens=%q", tokens)
		}
	}
}

func TestSplitWordsBytesAndRunes(t *testing.T) {
	words := NewScanner(bytes.NewBufferString("  alpha\tbeta\nlast"))
	words.Split(ScanWords)
	got, err := collect(words)
	if err != nil || strings.Join(got, ",") != "alpha,beta,last" {
		t.Fatalf("words=%q err=%v", got, err)
	}

	runes := NewScanner(bytes.NewBufferString("a€世"))
	runes.Split(ScanRunes)
	got, err = collect(runes)
	if err != nil || strings.Join(got, ",") != "a,€,世" {
		t.Fatalf("runes=%q err=%v", got, err)
	}
}

func TestBufferRaisesTokenLimit(t *testing.T) {
	line := strings.Repeat("x", MaxScanTokenSize+10)
	small := NewScanner(bytes.NewBufferString(line))
	if small.Scan() || small.Err() == nil {
		t.Fatal("default scanner accepted oversized token")
	}

	large := NewScanner(bytes.NewBufferString(line))
	large.Buffer(make([]byte, 32), len(line)+1)
	if !large.Scan() || large.Text() != line || large.Err() != nil {
		t.Fatalf("large scan len=%d err=%v", len(large.Text()), large.Err())
	}
}

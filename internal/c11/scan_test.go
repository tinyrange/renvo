package c11

import "testing"

func TestScanIndexesMatchingDelimitersAcrossTokenSlices(t *testing.T) {
	source := []byte("call(alpha[beta], (gamma + delta))")
	result := scan(source)
	if !result.ok {
		t.Fatal("scan failed")
	}
	open := -1
	for i := 0; i < len(result.tokens); i++ {
		if tokenIs(source, result.tokens[i], "(") {
			open = i
			break
		}
	}
	if open < 0 {
		t.Fatal("opening delimiter not found")
	}
	close := matchingToken(source, result.tokens, open, "(", ")")
	if close <= open || !tokenIs(source, result.tokens[close], ")") {
		t.Fatalf("full-token match = %d, want closing parenthesis after %d", close, open)
	}
	slice := result.tokens[open:]
	if got := matchingToken(source, slice, 0, "(", ")"); got != close-open {
		t.Fatalf("sliced-token match = %d, want %d", got, close-open)
	}
}

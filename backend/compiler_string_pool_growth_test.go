package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestStringDataPoolRetainsDistantMatchesAndResolvesCollisions(t *testing.T) {
	var g renvoLinearGen
	first := renvoAddStringData(&g, []byte("Aa"))
	collision := renvoAddStringData(&g, []byte("B@"))
	if renvoStringDataHash([]byte("Aa")) != renvoStringDataHash([]byte("B@")) || first == collision {
		t.Fatal("hash collision changed string identity")
	}
	var offsets []int
	for i := 0; i < 300; i++ {
		offsets = append(offsets, renvoAddStringData(&g, []byte(fmt.Sprintf("literal-%d\x00tail", i))))
	}
	before := len(g.asm.data)
	for i, offset := range offsets {
		value := []byte(fmt.Sprintf("literal-%d\x00tail", i))
		if got := renvoAddStringData(&g, value); got != offset || !bytes.Equal(g.asm.data[got:got+len(value)], value) {
			t.Fatalf("literal %d lost after growth", i)
		}
	}
	if renvoAddStringData(&g, []byte("Aa")) != first || renvoAddStringData(&g, []byte("B@")) != collision || len(g.asm.data) != before {
		t.Fatal("distant repeated strings were emitted again")
	}
}

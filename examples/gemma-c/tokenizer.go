package main

import (
	"fmt"
	"os"
	"unicode/utf8"
)

// Tables stay packed: no runtime JSON parser, maps, or vocabulary strings.
var tokData []byte
var tokHeader [16]int
var textMode bool
var bytePending []byte

func tokWord(at int) int {
	return int(uint32(tokData[at]) | uint32(tokData[at+1])<<8 | uint32(tokData[at+2])<<16 | uint32(tokData[at+3])<<24)
}
func tokFail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(2)
}
func tokLoad(path string) {
	var err error
	tokData, err = os.ReadFile(path)
	if err != nil || len(tokData) < 64 || len(tokData) > 64*1024*1024 {
		tokFail("Cannot load tokenizer tables")
	}
	for i := 0; i < 16; i++ {
		tokHeader[i] = tokWord(i * 4)
	}
	h := tokHeader
	if h[0] != 0x31544d47 || h[1] != 1 || h[2] != len(tokData) || h[3] < 262144 || h[3] > 262145 || h[4] < 1 || h[4] > 1<<21 || h[4]&(h[4]-1) != 0 || h[5] < 1 || h[5] > 1000000 {
		tokFail("Invalid tokenizer header")
	}
	if h[6] != 64 || h[7] != h[6]+0x110000*4 || h[8] != h[7]+h[4]*16 || h[9] != h[8]+h[5]*16 || h[10] != h[9]+(h[3]+1)*4 || h[11] != h[10]+h[3] || h[12] < h[11] || h[12]+1024 != len(tokData) {
		tokFail("Invalid tokenizer sections")
	}
	previous := 0
	for i := 0; i <= h[3]; i++ {
		offset := tokWord(h[9] + i*4)
		if offset < previous || offset > h[12]-h[11] {
			tokFail("Invalid token offset")
		}
		previous = offset
	}
	for i := 0; i < h[5]; i++ {
		at := h[8] + i*16
		child, sibling := tokWord(at), tokWord(at+4)
		if child != 0 && (child <= i || child >= h[5]) || sibling != 0 && sibling >= i || tokWord(at+8) > 255 || tokWord(at+12) > h[3] {
			tokFail("Invalid added-token trie")
		}
	}
}

func tokPair(a int, b int) (int, int) {
	mask := uint32(tokHeader[4] - 1)
	slot := ((uint32(a) * 0x9e3779b1) ^ (uint32(b) * 0x85ebca77)) & mask
	for attempts := 0; attempts < tokHeader[4]; attempts++ {
		at := tokHeader[7] + int(slot)*16
		rank := tokWord(at + 8)
		if rank == 0 {
			return -1, -1
		}
		if tokWord(at) == a && tokWord(at+4) == b {
			id := tokWord(at + 12)
			if id >= tokHeader[3] {
				tokFail("Invalid merge token")
			}
			return rank - 1, id
		}
		slot = (slot + 1) & mask
	}
	tokFail("Invalid merge table")
	return -1, -1
}

type tokMerge struct{ rank, pos, id int }
type tokWork struct {
	ids, prev, next []int
	heap            []tokMerge
}

func tokLess(a tokMerge, b tokMerge) bool {
	return a.rank < b.rank || a.rank == b.rank && a.pos < b.pos
}
func tokPush(w *tokWork, pos int) {
	if pos < 0 || w.ids[pos] < 0 || w.next[pos] < 0 {
		return
	}
	rank, id := tokPair(w.ids[pos], w.ids[w.next[pos]])
	if rank < 0 {
		return
	}
	entry := tokMerge{rank: rank, pos: pos, id: id}
	w.heap = append(w.heap, entry)
	i := len(w.heap) - 1
	for i > 0 {
		parent := (i - 1) / 2
		if !tokLess(entry, w.heap[parent]) {
			break
		}
		w.heap[i] = w.heap[parent]
		i = parent
	}
	w.heap[i] = entry
}
func tokPop(w *tokWork) tokMerge {
	result := w.heap[0]
	last := w.heap[len(w.heap)-1]
	w.heap = w.heap[:len(w.heap)-1]
	if len(w.heap) == 0 {
		return result
	}
	i := 0
	for i*2+1 < len(w.heap) {
		child := i*2 + 1
		if child+1 < len(w.heap) && tokLess(w.heap[child+1], w.heap[child]) {
			child++
		}
		if !tokLess(w.heap[child], last) {
			break
		}
		w.heap[i] = w.heap[child]
		i = child
	}
	w.heap[i] = last
	return result
}

func tokBPE(text string, output []int) []int {
	w := tokWork{}
	for at := 0; at < len(text); {
		r, size := utf8.DecodeRuneInString(text[at:])
		if r == utf8.RuneError && size == 1 {
			tokFail("Prompt must be valid UTF-8")
		}
		if r == ' ' {
			r = '▁'
		}
		id := tokWord(tokHeader[6]+int(r)*4) - 1
		if id >= tokHeader[3] {
			tokFail("Invalid character token")
		}
		if id >= 0 {
			w.ids = append(w.ids, id)
		} else {
			for j := 0; j < size; j++ {
				id = tokWord(tokHeader[12] + int(text[at+j])*4)
				if id >= tokHeader[3] {
					tokFail("Invalid byte token")
				}
				w.ids = append(w.ids, id)
			}
		}
		at += size
	}
	w.prev = make([]int, len(w.ids))
	w.next = make([]int, len(w.ids))
	for i := 0; i < len(w.ids); i++ {
		w.prev[i] = i - 1
		w.next[i] = i + 1
	}
	if len(w.ids) > 0 {
		w.next[len(w.ids)-1] = -1
	}
	for i := 0; i < len(w.ids); i++ {
		tokPush(&w, i)
	}
	for len(w.heap) > 0 {
		entry := tokPop(&w)
		left := entry.pos
		right := w.next[left]
		if w.ids[left] < 0 || right < 0 {
			continue
		}
		rank, id := tokPair(w.ids[left], w.ids[right])
		if rank != entry.rank || id != entry.id {
			continue
		}
		w.ids[left] = id
		w.ids[right] = -1
		w.next[left] = w.next[right]
		if w.next[left] >= 0 {
			w.prev[w.next[left]] = left
		}
		tokPush(&w, w.prev[left])
		tokPush(&w, left)
	}
	for i := 0; i < len(w.ids); i++ {
		if w.ids[i] >= 0 {
			output = append(output, w.ids[i])
		}
	}
	return output
}

// Added tokens are matched before normalization: leftmost, longest match.
func tokAdded(text string, start int) (int, int) {
	node, id, end := 0, -1, start
	for at := start; at < len(text); at++ {
		child := tokWord(tokHeader[8] + node*16)
		for child != 0 && tokWord(tokHeader[8]+child*16+8) != int(text[at]) {
			child = tokWord(tokHeader[8] + child*16 + 4)
		}
		if child == 0 {
			break
		}
		node = child
		terminal := tokWord(tokHeader[8] + node*16 + 12)
		if terminal != 0 {
			id = terminal - 1
			end = at + 1
		}
	}
	return id, end
}
func tokEncode(text string) []int {
	if len(text) > 65536 {
		tokFail("Prompt exceeds 65536 UTF-8 bytes")
	}
	output := make([]int, 0)
	start := 0
	for at := 0; at < len(text); {
		id, end := tokAdded(text, at)
		if id < 0 {
			at++
			continue
		}
		output = tokBPE(text[start:at], output)
		output = append(output, id)
		at = end
		start = end
	}
	return tokBPE(text[start:], output)
}

func tokSpace(r rune) bool {
	// Python/Jinja str.strip whitespace, including U+001C..U+001F.
	return r >= 9 && r <= 13 || r >= 28 && r <= 32 || r == 0x85 || r == 0xa0 || r == 0x1680 || r >= 0x2000 && r <= 0x200a || r == 0x2028 || r == 0x2029 || r == 0x202f || r == 0x205f || r == 0x3000
}
func tokChat(text string) []int {
	start, end := 0, 0
	for at := 0; at < len(text); {
		r, size := utf8.DecodeRuneInString(text[at:])
		if r == utf8.RuneError && size == 1 {
			tokFail("Prompt must be valid UTF-8")
		}
		if !tokSpace(r) {
			end = at + size
		} else if at == start {
			start = at + size
		}
		at += size
	}
	if end < start {
		end = start
	}
	return tokEncode("<bos><start_of_turn>user\n" + text[start:end] + "<end_of_turn>\n<start_of_turn>model\n")
}

func tokFlushBytes() {
	if len(bytePending) == 0 {
		return
	}
	text := string(bytePending)
	valid := true
	for at := 0; at < len(text); {
		r, size := utf8.DecodeRuneInString(text[at:])
		if r == utf8.RuneError && size == 1 {
			valid = false
			break
		}
		at += size
	}
	if valid {
		fmt.Print(text)
	} else {
		// Hugging Face ByteFallback replaces every byte in an invalid run.
		for i := 0; i < len(bytePending); i++ {
			fmt.Print("�")
		}
	}
	bytePending = bytePending[:0]
}
func tokEmit(id int) {
	if id < 0 || id >= tokHeader[3] {
		tokFail("Invalid output token ID")
	}
	flags := tokData[tokHeader[10]+id]
	if flags&1 != 0 {
		return
	}
	start := tokHeader[11] + tokWord(tokHeader[9]+id*4)
	end := tokHeader[11] + tokWord(tokHeader[9]+(id+1)*4)
	if flags&2 != 0 {
		bytePending = append(bytePending, tokData[start:end]...)
		return
	}
	tokFlushBytes()
	fmt.Print(string(tokData[start:end]))
}

package driver

import (
	"math/rand"
	"testing"
)

func TestSourceEmbedArchiveMatchPreservesSelection(t *testing.T) {
	random := rand.New(rand.NewSource(1))
	for mode := 0; mode < 5; mode++ {
		data := make([]byte, 5000+mode)
		for i := range data {
			switch mode {
			case 0:
				data[i] = 'a'
			case 1:
				data[i] = byte(i % 7)
			case 2:
				data[i] = byte(random.Intn(5))
			case 3:
				data[i] = byte(random.Intn(256))
			case 4:
				data[i] = byte((i/19 + i%3) % 13)
			}
		}
		buckets := make([]int32, 65536)
		previous := make([]int32, len(data))
		for pos := 0; pos < len(data); pos++ {
			distance, length := sourceEmbedArchiveMatch(data, buckets, previous, pos)
			wantDistance, wantLength := referenceSourceEmbedArchiveMatch(data, buckets, previous, pos)
			if distance != wantDistance || length != wantLength {
				t.Fatalf("mode=%d pos=%d got=(%d,%d) want=(%d,%d)", mode, pos, distance, length, wantDistance, wantLength)
			}
			sourceEmbedArchiveAddPosition(data, buckets, previous, pos)
		}
	}
}

func referenceSourceEmbedArchiveMatch(data []byte, buckets []int32, previous []int32, pos int) (int, int) {
	// Bound search work even on adversarial buckets. The best-match boundary
	// check below avoids rescanning shared prefixes during the deeper search.
	const maxCandidates = 256
	const maxLength = 273
	if pos+2 >= len(data) {
		return 0, 0
	}
	bestDistance := 0
	bestLength := 0
	checked := 0
	bucket := sourceEmbedArchiveBucket(data, pos, len(buckets))
	for candidate := int(buckets[bucket]) - 1; candidate >= 0 && checked < maxCandidates; candidate = int(previous[candidate]) - 1 {
		distance := pos - candidate
		if distance > 4096 {
			break
		}
		checked++
		// A candidate must extend the current best match to improve it. Reject
		// mismatches at that boundary before rescanning an identical prefix.
		// This keeps the deeper, size-saving search cheap on repetitive source.
		if bestLength > 0 && pos+bestLength < len(data) && data[candidate+bestLength] != data[pos+bestLength] {
			continue
		}
		if data[candidate] != data[pos] || data[candidate+1] != data[pos+1] || data[candidate+2] != data[pos+2] {
			continue
		}
		length := 0
		for length < maxLength && pos+length < len(data) && data[candidate+length] == data[pos+length] {
			length++
		}
		if length > bestLength {
			bestDistance = distance
			bestLength = length
			if length == maxLength {
				break
			}
		}
	}
	return bestDistance, bestLength
}

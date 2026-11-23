package app

import (
	"strings"
	"testing"
)

// TestShuffledStringMapIsPermutation verifies the helper returns all keys in
// some order and actually randomizes between runs.
func TestShuffledStringMapIsPermutation(t *testing.T) {
	data := map[string]string{}
	for i := 0; i < 20; i++ {
		data[string(rune('ぁ'+i))] = string(rune('a' + i))
	}
	first := shuffledStringMap(data)
	if len(first) != len(data) {
		t.Fatalf("shuffledStringMap lost keys: %d vs %d", len(first), len(data))
	}
	seen := map[string]bool{}
	for _, k := range first {
		if seen[k] {
			t.Fatalf("duplicate key %q in shuffled output", k)
		}
		seen[k] = true
	}
	// With 20 keys, getting the identical order repeatedly would be vanishingly
	// unlikely; check at least one of several runs differs from sorted order.
	sorted := sortedStringMap(data)
	differs := false
	for run := 0; run < 10 && !differs; run++ {
		got := shuffledStringMap(data)
		if strings.Join(got, ",") != strings.Join(sorted, ",") {
			differs = true
		}
	}
	if !differs {
		t.Errorf("shuffledStringMap always returns sorted order")
	}
}

package tokenmatcher

import (
	"testing"
)

func testBestMatch(t *testing.T, reference string, candidates []string, requiredResult *TokenizedSource) {
	referenceTS := NewTokenizedSource(reference)
	candidatesTS := make([]*TokenizedSource, len(candidates))
	for i, candidate := range candidates {
		candidatesTS[i] = NewTokenizedSource(candidate)
	}
	bestMatch := FindBestMatch(referenceTS, candidatesTS)
	if !requiredResult.Equals(bestMatch) {
		t.Errorf("best Match should have been '%s', was '%s'", requiredResult, bestMatch)
	}
}

func Test(t *testing.T) {
	testBestMatch(t, "a:b:c:d", []string{"a", "a:b", "a:*:c"}, NewTokenizedSource("a:*:c"))
	testBestMatch(t, "a:b:c:d", []string{"a", "a:x", "a:*:y"}, NewTokenizedSource("a"))
	testBestMatch(t, "a:b:c:d", []string{"a", "a:b"}, NewTokenizedSource("a:b"))
	testBestMatch(t, "a:b:c:d", []string{"x", "x:y"}, nil)
	testBestMatch(t, "a:b:c:d", []string{}, nil)
	testBestMatch(t, "a:b:c:d", []string{"a:b:c:d"}, NewTokenizedSource("a:b:c:d"))
	testBestMatch(t, "a:b:c:d", []string{"a:b:c", "a:b:c:d"}, NewTokenizedSource("a:b:c:d"))
	testBestMatch(t, "a:b:c:d", []string{"a:b:c", "a:b:c:d", "a:b:*:d"}, NewTokenizedSource("a:b:c:d"))
	testBestMatch(t, "a:b:c:d", []string{"a:b:*:d", "a:b:c", "a:b:c:d", "a:b:*:d"}, NewTokenizedSource("a:b:c:d"))
	testBestMatch(t, "a:b:c:d", []string{"a:b:*:d", "a:b:c", "a:b:c:d", "a:b:*:d"}, NewTokenizedSource("a:b:c:d"))
	testBestMatch(t, "a:b:c:d", []string{"a:b:*:d", "a:*:c:d"}, NewTokenizedSource("a:b:*:d"))
	testBestMatch(t, "a:b:c:d", []string{"a:b:c:d", "a:b:*:d"}, NewTokenizedSource("a:b:c:d"))
	testBestMatch(t, "a:b:c:d", []string{"a:b:*:d", "a:b:c:d"}, NewTokenizedSource("a:b:c:d"))
	testBestMatch(t, "a:b:c:d", []string{"a:b:*:d", "a:*:c:d"}, NewTokenizedSource("a:b:*:d"))
	testBestMatch(t, "a:b:c:d", []string{"a:*:c:d", "a:b:*:d"}, NewTokenizedSource("a:b:*:d"))
	testBestMatch(t, "a:b:c:d", []string{"a:*:c:d", "a:*:*:d"}, NewTokenizedSource("a:*:c:d"))
	testBestMatch(t, "a:b:c:d", []string{"a:*:*:d", "a:*:c:d"}, NewTokenizedSource("a:*:c:d"))
	testBestMatch(t, "a:b:c:d", []string{"a:*:*:d", "a:b:c"}, NewTokenizedSource("a:*:*:d"))
	testBestMatch(t, "a:b:c:d", []string{"a:b:c", "a:*:*:d"}, NewTokenizedSource("a:*:*:d"))
	testBestMatch(t, "a:b:c:d", []string{"a:b:c", "a:*:c"}, NewTokenizedSource("a:b:c"))
	testBestMatch(t, "a:b:c:d", []string{"a:*:c", "a:b:c"}, NewTokenizedSource("a:b:c"))
	testBestMatch(t, "a:b:c:d", []string{"a:b", "a:*:c"}, NewTokenizedSource("a:*:c"))
	testBestMatch(t, "a:b:c:d", []string{"a:*:c", "a:b"}, NewTokenizedSource("a:*:c"))
	testBestMatch(t, "a:b:c:d", []string{"a", "a:b"}, NewTokenizedSource("a:b"))
	testBestMatch(t, "a:b:c:d", []string{"a:b", "a"}, NewTokenizedSource("a:b"))
}

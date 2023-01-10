package tokenmatcher

import "strings"

// Does not do much, we mostly have this to make the code more readable (better than 2D string arrays)
type TokenizedSource struct {
	Tokens []string
}

func (this TokenizedSource) String() string {
	return strings.Join(this.Tokens, ":")
}

func (this *TokenizedSource) Equals(other *TokenizedSource) bool {
	if this == nil && other == nil {
		return true
	}
	if this == nil || other == nil {
		return false
	}
	return this.String() == other.String()
}

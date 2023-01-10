package tokenmatcher

import (
	"strings"
)

func NewTokenizedSource(source string) *TokenizedSource {
	return &TokenizedSource{
		Tokens: strings.Split(source, ":"),
	}
}

func generatePatterns(reference *TokenizedSource) []*TokenizedSource {
	if len(reference.Tokens) > 10 {
		panic("No more than 10 tokens supported. More tokens lead to an explosion of possible wildcard positions")
	}

	var patterns []*TokenizedSource

	// start with all the tokens and use one less with every iteration
	for i := len(reference.Tokens); i > 0; i-- {
		limitedTokens := reference.Tokens[0:i]

		patterns = append(patterns, &TokenizedSource{Tokens: limitedTokens})
		if len(limitedTokens) > 2 {
			// we're setting the wildcards as if we were counting. 'j' is our counter, starting at 1 (one single
			// wildcard at the rightmost allowed position)
			for j := 1; j < (1 << (len(limitedTokens) - 2)); j++ {
				// create copy of limitedTokens. We can't modify limitedTokens directly.
				var wildcardedTokens []string
				wildcardedTokens = append(wildcardedTokens, limitedTokens...)

				// 'p' goes through all the bits of 'j' and checks if they are set. If yes, it places a wildcard.
				for p := 0; p < len(wildcardedTokens)-2; p++ {
					if j&(1<<p) > 0 {
						wildcardedTokens[len(wildcardedTokens)-2-p] = "*"
					}
				}
				patterns = append(patterns, &TokenizedSource{Tokens: wildcardedTokens})
			}
		}
	}

	return patterns
}

func FindBestMatch(reference *TokenizedSource, candidates []*TokenizedSource) *TokenizedSource {
	patterns := generatePatterns(reference)
	for _, pattern := range patterns {
		for _, candidate := range candidates {
			if candidate.String() == pattern.String() {
				return candidate
			}
		}
	}

	return nil
}

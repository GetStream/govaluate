package govaluate

import (
	"testing"
)

/*
	Tests to make sure that all the different token kinds have different string representations
	Gotta get that 95% code coverage yall. That's why tests like this get written; over-reliance on bad metrics.
*/
func TestTokenKindStrings(test *testing.T) {
	kinds := []TokenKind{
		UNKNOWN,
		PREFIX,
		NUMERIC,
		BOOLEAN,
		STRING,
		PATTERN,
		TIME,
		VARIABLE,
		COMPARATOR,
		LOGICALOP,
		MODIFIER,
		CLAUSE,
		CLAUSE_CLOSE,
		TERNARY,
	}

	kindStrings := make(map[string]struct{}, len(kinds))
	for _, kind := range kinds {
		s := kind.String()
		if _, ok := kindStrings[s]; ok {
			test.Logf("Token kind test found duplicate string for token kind %v ('%v')\n", kind, s)
			test.FailNow()
		}
		kindStrings[s] = struct{}{}
	}
}

package govaluate

type tokenStream struct {
	tokens      []ExpressionToken
	index       int
	tokenLength int
}

func newTokenStream(tokens []ExpressionToken) *tokenStream {
	return &tokenStream{
		tokens:      tokens,
		tokenLength: len(tokens),
	}
}

func (s *tokenStream) rewind() {
	s.index--
}

func (s *tokenStream) next() ExpressionToken {
	t := s.tokens[s.index]
	s.index++
	return t
}

func (s tokenStream) hasNext() bool {
	return s.index < s.tokenLength
}

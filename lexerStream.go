package govaluate

type lexerStream struct {
	source   []rune
	position int
	length   int
}

func newLexerStream(source string) *lexerStream {
	runes := []rune(source)
	return &lexerStream{
		source: runes,
		length: len(runes),
	}
}

func (s *lexerStream) readCharacter() rune {
	c := s.source[s.position]
	s.position++
	return c
}

func (s *lexerStream) rewind(amount int) {
	s.position -= amount
}

func (s lexerStream) canRead() bool {
	return s.position < s.length
}

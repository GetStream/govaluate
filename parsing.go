package govaluate

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func parseTokens(expression string, functions map[string]ExpressionFunction) ([]ExpressionToken, error) {

	var ret []ExpressionToken
	var token ExpressionToken
	var stream *lexerStream
	var state lexerState
	var err error
	var found bool

	stream = newLexerStream(expression)
	state = validLexerStates[0]

	for stream.canRead() {

		token, found, err = readToken(stream, state, functions)
		if err != nil {
			return ret, err
		} else if !found {
			break
		}

		state, err = getLexerStateForToken(token.Kind)
		if err != nil {
			return ret, err
		}

		// append this valid token
		ret = append(ret, token)
	}

	if err := checkBalance(ret); err != nil {
		return nil, err
	}
	return ret, nil
}

//nolint: gocognit
func readToken(stream *lexerStream, state lexerState, functions map[string]ExpressionFunction) (ExpressionToken, bool, error) {

	var function ExpressionFunction
	var ret ExpressionToken
	var tokenValue interface{}
	var tokenTime time.Time
	var tokenString string
	var kind TokenKind
	var character rune
	var found bool
	var completed bool
	var err error

	// numeric is 0-9, or . or 0x followed by digits
	// string starts with '
	// variable is alphanumeric, always starts with a letter
	// bracket always means variable
	// symbols are anything non-alphanumeric
	// all others read into a buffer until they reach the end of the stream
	for stream.canRead() {

		character = stream.readCharacter()

		if unicode.IsSpace(character) {
			continue
		}

		kind = UNKNOWN //nolint: ineffassign

		// numeric constant
		if isNumeric(character) {

			if stream.canRead() && character == '0' {
				character = stream.readCharacter()

				if stream.canRead() && character == 'x' {
					tokenString, _ = readUntilFalse(stream, false, true, isHexDigit)
					tokenValueInt, err := strconv.ParseUint(tokenString, 16, 64)

					if err != nil {
						return ExpressionToken{}, false,
							fmt.Errorf("Unable to parse hex value '%v' to uint64", tokenString)
					}

					kind = NUMERIC
					tokenValue = float64(tokenValueInt)
					break
				} else {
					stream.rewind(1)
				}
			}

			tokenString = readTokenUntilFalse(stream, isNumeric)
			tokenValue, err = strconv.ParseFloat(tokenString, 64)

			if err != nil {
				return ExpressionToken{}, false,
					fmt.Errorf("Unable to parse numeric value '%v' to float64", tokenString)
			}
			kind = NUMERIC
			break
		}

		// comma, separator
		if character == ',' {

			tokenValue = ","
			kind = SEPARATOR
			break
		}

		// escaped variable
		if character == '[' {

			tokenValue, completed = readUntilFalse(stream, true, false, isNotClosingBracket)
			kind = VARIABLE

			if !completed {
				return ExpressionToken{}, false, errors.New("Unclosed parameter bracket")
			}

			// above method normally rewinds us to the closing bracket, which we want to skip.
			stream.rewind(-1)
			break
		}

		// regular variable - or function?
		if unicode.IsLetter(character) {

			tokenString = readTokenUntilFalse(stream, isVariableName)

			kind, tokenValue = VARIABLE, tokenString

			// boolean?
			if tokenValue == "true" {
				kind, tokenValue = BOOLEAN, true
			} else if tokenValue == "false" {
				kind, tokenValue = BOOLEAN, false
			}

			// textual operator?
			if tokenValue == "in" || tokenValue == "IN" {

				// force lower case for consistency
				kind, tokenValue = COMPARATOR, "in"
			}

			// function?
			function, found = functions[tokenString]
			if found {
				kind, tokenValue = FUNCTION, function
			}

			// accessor?
			accessorIndex := strings.Index(tokenString, ".")
			if accessorIndex > 0 {

				// check that it doesn't end with a hanging period
				if tokenString[len(tokenString)-1] == '.' {
					return ExpressionToken{}, false, fmt.Errorf("Hanging accessor on token '%s'", tokenString)
				}

				kind = ACCESSOR
				splits := strings.Split(tokenString, ".")
				tokenValue = splits

				// check that none of them are unexported
				for i := 1; i < len(splits); i++ {

					firstCharacter := getFirstRune(splits[i])

					if unicode.ToUpper(firstCharacter) != firstCharacter {
						return ExpressionToken{}, false,
							fmt.Errorf("Unable to access unexported field '%s' in token '%s'", splits[i], tokenString)
					}
				}
			}
			break
		}

		if !isNotQuote(character) {
			tokenValue, completed = readUntilFalse(stream, true, false, isNotQuote)

			if !completed {
				return ExpressionToken{}, false, errors.New("Unclosed string literal")
			}

			// advance the stream one position, since reading until false assumes the terminator is a real token
			stream.rewind(-1)

			// check to see if this can be parsed as a time.
			tokenTime, found = tryParseTime(tokenValue.(string))
			if found {
				kind = TIME
				tokenValue = tokenTime
			} else {
				kind = STRING
			}
			break
		}

		if character == '(' {
			tokenValue = character
			kind = CLAUSE
			break
		}

		if character == ')' {
			tokenValue = character
			kind = CLAUSE_CLOSE
			break
		}

		// must be a known symbol
		tokenString = readTokenUntilFalse(stream, isNotAlphanumeric)
		tokenValue = tokenString

		// quick hack for the case where "-" can mean "prefixed negation" or "minus", which are used
		// very differently.
		if state.canTransitionTo(PREFIX) {
			_, found = prefixSymbols[tokenString]
			if found {

				kind = PREFIX
				break
			}
		}
		_, found = modifierSymbols[tokenString]
		if found {

			kind = MODIFIER
			break
		}

		_, found = logicalSymbols[tokenString]
		if found {

			kind = LOGICALOP
			break
		}

		_, found = comparatorSymbols[tokenString]
		if found {

			kind = COMPARATOR
			break
		}

		_, found = ternarySymbols[tokenString]
		if found {

			kind = TERNARY
			break
		}

		return ret, false, fmt.Errorf("Invalid token: '%s'", tokenString)
	}

	ret.Kind = kind
	ret.Value = tokenValue

	return ret, (kind != UNKNOWN), nil
}

func readTokenUntilFalse(stream *lexerStream, condition func(rune) bool) string {

	var ret string

	stream.rewind(1)
	ret, _ = readUntilFalse(stream, false, true, condition)
	return ret
}

/*
	Returns the string that was read until the given [condition] was false, or whitespace was broken.
	Returns false if the stream ended before whitespace was broken or condition was met.
*/
func readUntilFalse(stream *lexerStream, includeWhitespace, breakWhitespace bool, condition func(rune) bool) (string, bool) {

	var tokenBuffer bytes.Buffer
	var character rune
	var conditioned bool

	conditioned = false

	for stream.canRead() {

		character = stream.readCharacter()

		// Use backslashes to escape anything
		if character == '\\' {

			character = stream.readCharacter()
			tokenBuffer.WriteString(string(character))
			continue
		}

		if unicode.IsSpace(character) {

			if breakWhitespace && tokenBuffer.Len() > 0 {
				conditioned = true
				break
			}
			if !includeWhitespace {
				continue
			}
		}

		if condition(character) {
			tokenBuffer.WriteString(string(character))
		} else {
			conditioned = true
			stream.rewind(1)
			break
		}
	}

	return tokenBuffer.String(), conditioned
}

/*
	Checks to see if any optimizations can be performed on the given [tokens], which form a complete, valid expression.
	The returns slice will represent the optimized (or unmodified) list of tokens to use.
*/
func optimizeTokens(tokens []ExpressionToken) ([]ExpressionToken, error) {

	var token ExpressionToken
	var symbol OperatorSymbol
	var err error
	var index int

	for index, token = range tokens {

		// if we find a regex operator, and the right-hand value is a constant, precompile and replace with a pattern.
		if token.Kind != COMPARATOR {
			continue
		}

		symbol = comparatorSymbols[token.Value.(string)]
		if symbol != REQ && symbol != NREQ {
			continue
		}

		index++
		token = tokens[index]
		if token.Kind == STRING {

			token.Kind = PATTERN
			token.Value, err = regexp.Compile(token.Value.(string))

			if err != nil {
				return tokens, err
			}

			tokens[index] = token
		}
	}
	return tokens, nil
}

/*
	Checks the balance of tokens which have multiple parts, such as parenthesis.
*/
func checkBalance(tokens []ExpressionToken) error {
	var token ExpressionToken
	var parens int

	stream := newTokenStream(tokens)
	for stream.hasNext() {

		token = stream.next()
		if token.Kind == CLAUSE {
			parens++
			continue
		}
		if token.Kind == CLAUSE_CLOSE {
			parens--
			continue
		}
	}

	if parens != 0 {
		return errors.New("Unbalanced parenthesis")
	}
	return nil
}

func isHexDigit(character rune) bool {
	character = unicode.ToLower(character)
	return unicode.IsDigit(character) ||
		character == 'a' ||
		character == 'b' ||
		character == 'c' ||
		character == 'd' ||
		character == 'e' ||
		character == 'f'
}

func isNumeric(character rune) bool {
	return unicode.IsDigit(character) || character == '.'
}

func isNotQuote(character rune) bool {
	return character != '\'' && character != '"'
}

func isNotAlphanumeric(character rune) bool {

	return !(unicode.IsDigit(character) ||
		unicode.IsLetter(character) ||
		character == '(' ||
		character == ')' ||
		character == '[' ||
		character == ']' || // starting to feel like there needs to be an `isOperation` func (#59)
		!isNotQuote(character))
}

func isVariableName(character rune) bool {

	return unicode.IsLetter(character) ||
		unicode.IsDigit(character) ||
		character == '_' ||
		character == '.'
}

func isNotClosingBracket(character rune) bool {

	return character != ']'
}

/*
	Attempts to parse the [candidate] as a Time.
	Tries a series of standardized date formats, returns the Time if one applies,
	otherwise returns false through the second return.
*/
func tryParseTime(candidate string) (time.Time, bool) {

	var ret time.Time
	var found bool

	timeFormats := [...]string{
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		time.Kitchen,
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02",                         // RFC 3339
		"2006-01-02 15:04",                   // RFC 3339 with minutes
		"2006-01-02 15:04:05",                // RFC 3339 with seconds
		"2006-01-02 15:04:05-07:00",          // RFC 3339 with seconds and timezone
		"2006-01-02T15Z0700",                 // ISO8601 with hour
		"2006-01-02T15:04Z0700",              // ISO8601 with minutes
		"2006-01-02T15:04:05Z0700",           // ISO8601 with seconds
		"2006-01-02T15:04:05.999999999Z0700", // ISO8601 with nanoseconds
	}

	for _, format := range timeFormats {
		ret, found = tryParseExactTime(candidate, format)
		if found {
			return ret, true
		}
	}

	return time.Now(), false
}

func tryParseExactTime(candidate, format string) (time.Time, bool) {
	ret, err := time.ParseInLocation(format, candidate, time.Local)
	if err != nil {
		return time.Now(), false
	}

	return ret, true
}

func getFirstRune(candidate string) rune {
	for _, character := range candidate {
		return character
	}
	return 0
}

package lang

import "unicode/utf8"

// Lex tokenizes a single line of input into a slice of tokens.
func Lex(input string) []Token {
	var tokens []Token
	i := 0
	for i < len(input) {
		ch := input[i]

		// Skip whitespace
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			i++
			continue
		}

		switch ch {
		case '+':
			tokens = append(tokens, Token{Type: TOKEN_PLUS, Literal: "+", Pos: i})
			i++
		case '-':
			tokens = append(tokens, Token{Type: TOKEN_MINUS, Literal: "-", Pos: i})
			i++
		case '*':
			if i+1 < len(input) && input[i+1] == '*' {
				tokens = append(tokens, Token{Type: TOKEN_STARSTAR, Literal: "**", Pos: i})
				i += 2
			} else {
				tokens = append(tokens, Token{Type: TOKEN_STAR, Literal: "*", Pos: i})
				i++
			}
		case '&':
			tokens = append(tokens, Token{Type: TOKEN_AMP, Literal: "&", Pos: i})
			i++
		case '|':
			tokens = append(tokens, Token{Type: TOKEN_PIPE, Literal: "|", Pos: i})
			i++
		case '^':
			tokens = append(tokens, Token{Type: TOKEN_CARET, Literal: "^", Pos: i})
			i++
		case '~':
			tokens = append(tokens, Token{Type: TOKEN_TILDE, Literal: "~", Pos: i})
			i++
		case '!':
			tokens = append(tokens, Token{Type: TOKEN_BANG, Literal: "!", Pos: i})
			i++
		case '<':
			if i+1 < len(input) && input[i+1] == '<' {
				tokens = append(tokens, Token{Type: TOKEN_LSHIFT, Literal: "<<", Pos: i})
				i += 2
			} else {
				i++ // skip unknown <
			}
		case '>':
			if i+1 < len(input) && input[i+1] == '>' {
				tokens = append(tokens, Token{Type: TOKEN_RSHIFT, Literal: ">>", Pos: i})
				i += 2
			} else {
				i++ // skip unknown >
			}
		case '/':
			tokens = append(tokens, Token{Type: TOKEN_SLASH, Literal: "/", Pos: i})
			i++
		case '(':
			tokens = append(tokens, Token{Type: TOKEN_LPAREN, Literal: "(", Pos: i})
			i++
		case ')':
			tokens = append(tokens, Token{Type: TOKEN_RPAREN, Literal: ")", Pos: i})
			i++
		case '=':
			tokens = append(tokens, Token{Type: TOKEN_EQUALS, Literal: "=", Pos: i})
			i++
		case '.':
			tokens = append(tokens, Token{Type: TOKEN_DOT, Literal: ".", Pos: i})
			i++
		case '#':
			tokens = append(tokens, Token{Type: TOKEN_HASH, Literal: "#", Pos: i})
			i++
		case ',':
			tokens = append(tokens, Token{Type: TOKEN_COMMA, Literal: ",", Pos: i})
			i++
		case '%':
			tokens = append(tokens, Token{Type: TOKEN_PERCENT, Literal: "%", Pos: i})
			i++
		case '$':
			tokens = append(tokens, Token{Type: TOKEN_CURRENCY, Literal: "$", Pos: i})
			i++
		case '@':
			if end, ok := tryLexAt(input, i); ok {
				tokens = append(tokens, Token{Type: TOKEN_AT, Literal: input[i:end], Pos: i})
				i = end
			} else {
				i++ // skip unknown @
			}
		default:
			if isDigit(ch) {
				start := i
				// Check for 0x, 0b, 0o prefixed literals
				if ch == '0' && i+1 < len(input) {
					next := input[i+1]
					if next == 'x' || next == 'X' {
						i += 2 // skip "0x"
						for i < len(input) && isHexDigit(input[i]) {
							i++
						}
						tokens = append(tokens, Token{Type: TOKEN_NUMBER, Literal: input[start:i], Pos: start})
						continue
					}
					if next == 'b' || next == 'B' {
						i += 2 // skip "0b"
						for i < len(input) && (input[i] == '0' || input[i] == '1') {
							i++
						}
						tokens = append(tokens, Token{Type: TOKEN_NUMBER, Literal: input[start:i], Pos: start})
						continue
					}
					if next == 'o' || next == 'O' {
						i += 2 // skip "0o"
						for i < len(input) && input[i] >= '0' && input[i] <= '7' {
							i++
						}
						tokens = append(tokens, Token{Type: TOKEN_NUMBER, Literal: input[start:i], Pos: start})
						continue
					}
				}
				for i < len(input) && isDigit(input[i]) {
					i++
				}
				numStr := input[start:i]
				// Check for time literal: 1-2 digit number followed by ':'
				if len(numStr) <= 2 && i < len(input) && input[i] == ':' {
					if end, ok := tryLexTime(input, start); ok {
						i = end
						tokens = append(tokens, Token{Type: TOKEN_TIME, Literal: input[start:end], Pos: start})
						continue
					}
				}
				tokens = append(tokens, Token{Type: TOKEN_NUMBER, Literal: numStr, Pos: start})
			} else if isWordStart(ch) {
				start := i
				for i < len(input) && isWordContinue(input[i]) {
					i++
				}
				tokens = append(tokens, Token{Type: TOKEN_WORD, Literal: input[start:i], Pos: start})
			} else {
				// Check for multi-byte currency symbols: €, £, ¥
				r, size := utf8.DecodeRuneInString(input[i:])
				if r == '€' || r == '£' || r == '¥' {
					tokens = append(tokens, Token{Type: TOKEN_CURRENCY, Literal: string(r), Pos: i})
					i += size
				} else {
					// Unknown character — skip it
					i += size
				}
			}
		}
	}
	tokens = append(tokens, Token{Type: TOKEN_EOF, Literal: "", Pos: i})
	return tokens
}

// tryLexAt checks if input starting at pos matches @YYYY-MM-DD[THH:MM:SS],
// @H:MM[:SS], or @DIGITS (unix timestamp).
// Returns (endPos, true) if matched, (0, false) otherwise.
func tryLexAt(input string, pos int) (int, bool) {
	i := pos + 1 // past @
	if i >= len(input) || !isDigit(input[i]) {
		return 0, false
	}
	// Count leading digits
	digitStart := i
	for i < len(input) && isDigit(input[i]) {
		i++
	}
	numDigits := i - digitStart
	afterDigits := i

	// 4 digits + '-' → try date: @YYYY-M(M)-D(D)[THH:MM:SS]
	if numDigits == 4 && afterDigits < len(input) && input[afterDigits] == '-' {
		j := afterDigits + 1 // past first -
		if j < len(input) && isDigit(input[j]) {
			j++ // first month digit
			if j < len(input) && isDigit(input[j]) {
				j++ // optional second month digit
			}
			if j < len(input) && input[j] == '-' {
				j++ // past second -
				if j < len(input) && isDigit(input[j]) {
					j++ // first day digit
					if j < len(input) && isDigit(input[j]) {
						j++ // optional second day digit
					}
					// Optional time: 'T' or ' ' followed by H(H):MM:SS
					if j < len(input) && (input[j] == 'T' || input[j] == ' ') {
						k := j + 1
						if k < len(input) && isDigit(input[k]) {
							k++ // first hour digit
							if k < len(input) && isDigit(input[k]) {
								k++ // optional second hour digit
							}
							if k+5 <= len(input) &&
								input[k] == ':' &&
								isDigit(input[k+1]) && isDigit(input[k+2]) &&
								input[k+3] == ':' &&
								isDigit(input[k+4]) && isDigit(input[k+5]) {
								k += 6
								j = k
								// Optional timezone offset: ' +NNNN' or ' -NNNN'
								if j+6 <= len(input) && input[j] == ' ' &&
									(input[j+1] == '+' || input[j+1] == '-') &&
									isDigit(input[j+2]) && isDigit(input[j+3]) &&
									isDigit(input[j+4]) && isDigit(input[j+5]) {
									j += 6
								}
							}
						}
					}
					return j, true
				}
			}
		}
		// Date pattern failed — fall through to unix fallback
	}

	// 1-2 digits + ':' → try time: @HH:MM[:SS]
	if numDigits <= 2 && afterDigits < len(input) && input[afterDigits] == ':' {
		j := afterDigits + 1 // past ':'
		if j+2 <= len(input) && isDigit(input[j]) && isDigit(input[j+1]) {
			j += 2 // past MM
			// Optional :SS
			if j < len(input) && input[j] == ':' &&
				j+3 <= len(input) && isDigit(input[j+1]) && isDigit(input[j+2]) {
				j += 3
			}
			return j, true
		}
		// Time pattern failed — fall through to unix fallback
	}

	// Fallback: plain digits → unix timestamp
	return afterDigits, true
}

// tryLexTime checks if the input starting at pos matches HH:MM or HH:MM:SS.
// The hour part (1-2 digits) has already been scanned.
// Returns (endPos, true) if matched, (0, false) otherwise.
func tryLexTime(input string, pos int) (int, bool) {
	i := pos
	// Skip hour digits (1-2)
	for i < len(input) && isDigit(input[i]) {
		i++
	}
	hourLen := i - pos
	if hourLen < 1 || hourLen > 2 {
		return 0, false
	}
	// Expect ':'
	if i >= len(input) || input[i] != ':' {
		return 0, false
	}
	i++ // past ':'
	// Expect exactly 2 digits for minutes
	if i+2 > len(input) || !isDigit(input[i]) || !isDigit(input[i+1]) {
		return 0, false
	}
	i += 2 // past MM

	// Optional :SS
	if i < len(input) && input[i] == ':' {
		if i+3 <= len(input) && isDigit(input[i+1]) && isDigit(input[i+2]) {
			i += 3 // past :SS
		}
	}

	return i, true
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isHexDigit(ch byte) bool {
	return isDigit(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func isWordStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isWordContinue(ch byte) bool {
	return isWordStart(ch) || isDigit(ch)
}

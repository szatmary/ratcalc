package main

import (
	"image/color"
	"ratcalc/app/lang"
	"strings"
)

// TokenKind represents the category of a syntax token.
type TokenKind int

const (
	TokenPlain TokenKind = iota
	TokenKeyword
	TokenString
	TokenNumber
	TokenComment
	TokenOperator
	TokenVariable
	TokenUnit
	TokenEquals
	TokenParen
)

// Token is a span of text with a syntax category.
type Token struct {
	Text string
	Kind TokenKind
}

// tokenColors maps token kinds to colors. Dark-theme oriented.
var tokenColors = map[TokenKind]color.NRGBA{
	TokenPlain:    {R: 0xD4, G: 0xD4, B: 0xD4, A: 0xFF}, // light gray
	TokenKeyword:  {R: 0x56, G: 0x9C, B: 0xD6, A: 0xFF}, // blue
	TokenString:   {R: 0xCE, G: 0x91, B: 0x78, A: 0xFF}, // orange
	TokenNumber:   {R: 0xB5, G: 0xCE, B: 0xA8, A: 0xFF}, // green
	TokenComment:  {R: 0x6A, G: 0x99, B: 0x55, A: 0xFF}, // dark green
	TokenOperator: {R: 0xD4, G: 0xD4, B: 0xD4, A: 0xFF}, // light gray
	TokenVariable: {R: 0x9C, G: 0xDB, B: 0xFE, A: 0xFF}, // light blue
	TokenUnit:     {R: 0x4E, G: 0xC9, B: 0xB0, A: 0xFF}, // teal
	TokenEquals:   {R: 0xD4, G: 0xD4, B: 0xD4, A: 0xFF}, // light gray
	TokenParen:    {R: 0xFF, G: 0xD7, B: 0x00, A: 0xFF}, // yellow
}

// TokenColor returns the color for a token kind.
func TokenColor(kind TokenKind) color.NRGBA {
	if c, ok := tokenColors[kind]; ok {
		return c
	}
	return tokenColors[TokenPlain]
}

// langTokenToHighlight maps a lang.TokenType to a highlight TokenKind.
func langTokenToHighlight(t lang.TokenType) TokenKind {
	switch t {
	case lang.TOKEN_NUMBER:
		return TokenNumber
	case lang.TOKEN_WORD:
		return TokenVariable
	case lang.TOKEN_PLUS, lang.TOKEN_MINUS, lang.TOKEN_STAR, lang.TOKEN_SLASH:
		return TokenOperator
	case lang.TOKEN_LPAREN, lang.TOKEN_RPAREN:
		return TokenParen
	case lang.TOKEN_EQUALS:
		return TokenEquals
	case lang.TOKEN_DOT:
		return TokenNumber
	case lang.TOKEN_AT:
		return TokenString // orange for @ literals
	case lang.TOKEN_TIME:
		return TokenString // orange for time literals
	default:
		return TokenPlain
	}
}

// Tokenize splits a line into highlighted tokens using the lang lexer.
func Tokenize(line string) []Token {
	if line == "" {
		return nil
	}

	langTokens := lang.Lex(line)
	var result []Token
	lastEnd := 0

	for _, lt := range langTokens {
		if lt.Type == lang.TOKEN_EOF {
			break
		}

		// Add any whitespace/gap before this token
		if lt.Pos > lastEnd {
			result = append(result, Token{
				Text: line[lastEnd:lt.Pos],
				Kind: TokenPlain,
			})
		}

		// Check if this WORD token is a known unit or keyword
		kind := langTokenToHighlight(lt.Type)
		if lt.Type == lang.TOKEN_WORD {
			if lt.Literal == "Now" || lt.Literal == "Date" || lt.Literal == "Time" || lt.Literal == "Unix" || lt.Literal == "unix" || lt.Literal == "to" || lt.Literal == "hex" || lt.Literal == "bin" || lt.Literal == "oct" || strings.EqualFold(lt.Literal, "AM") || strings.EqualFold(lt.Literal, "PM") {
				kind = TokenKeyword
			} else if lang.IsTimezone(lt.Literal) {
				kind = TokenKeyword
			} else if lang.LookupUnit(lt.Literal) != nil {
				kind = TokenUnit
			}
		}

		result = append(result, Token{
			Text: lt.Literal,
			Kind: kind,
		})

		lastEnd = lt.Pos + len(lt.Literal)
	}

	// Any trailing text
	if lastEnd < len(line) {
		result = append(result, Token{
			Text: line[lastEnd:],
			Kind: TokenPlain,
		})
	}

	return result
}

package lang

import "fmt"

// TokenType represents the type of a lexer token.
type TokenType int

const (
	TOKEN_NUMBER TokenType = iota
	TOKEN_WORD
	TOKEN_PLUS
	TOKEN_MINUS
	TOKEN_STAR
	TOKEN_SLASH
	TOKEN_LPAREN
	TOKEN_RPAREN
	TOKEN_EQUALS
	TOKEN_DOT
	TOKEN_HASH
	TOKEN_AT
	TOKEN_COMMA
	TOKEN_PERCENT
	TOKEN_TIME
	TOKEN_EOF
)

// Token represents a single lexer token.
type Token struct {
	Type    TokenType
	Literal string
	Pos     int // byte offset in the input
}

func (t Token) String() string {
	return fmt.Sprintf("Token(%d, %q, %d)", t.Type, t.Literal, t.Pos)
}

package lang

import (
	"math/big"
	"strconv"
	"strings"
)

// Parser holds the state for parsing a token stream.
type Parser struct {
	tokens []Token
	pos    int
}

// Parse parses a single line (given as a token slice) into an AST node.
// Returns nil for empty lines.
func Parse(tokens []Token) (Node, error) {
	if len(tokens) == 0 {
		return nil, nil
	}
	// Check if all tokens are EOF
	if len(tokens) == 1 && tokens[0].Type == TOKEN_EOF {
		return nil, nil
	}

	p := &Parser{tokens: tokens, pos: 0}

	// Detect assignment: WORD = expr
	eqIdx := findFirstEquals(tokens)
	if eqIdx >= 0 {
		return p.parseAssignment(eqIdx)
	}

	node, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Check for "to" conversion
	node, err = p.parseConversion(node)
	if err != nil {
		return nil, err
	}

	// Make sure we consumed everything (except EOF)
	if p.peek().Type != TOKEN_EOF {
		return nil, &EvalError{Msg: "unexpected token: " + p.peek().Literal}
	}

	return node, nil
}

// findFirstEquals finds the index of the first EQUALS token.
// Returns -1 if no valid assignment pattern (single WORD starting with a letter, then =).
func findFirstEquals(tokens []Token) int {
	if len(tokens) < 2 {
		return -1
	}
	// Assignment: WORD = expr, where WORD starts with a letter
	if tokens[0].Type != TOKEN_WORD || tokens[1].Type != TOKEN_EQUALS {
		return -1
	}
	// Variable name must start with a letter
	if len(tokens[0].Literal) == 0 || !isLetter(rune(tokens[0].Literal[0])) {
		return -1
	}
	return 1
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func (p *Parser) parseAssignment(eqIdx int) (Node, error) {
	name := p.tokens[0].Literal

	// Skip past the '='
	p.pos = eqIdx + 1

	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Check for "to" conversion on the RHS
	expr, err = p.parseConversion(expr)
	if err != nil {
		return nil, err
	}

	if p.peek().Type != TOKEN_EOF {
		return nil, &EvalError{Msg: "unexpected token after assignment: " + p.peek().Literal}
	}

	return &Assignment{Name: name, Expr: expr}, nil
}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TOKEN_EOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() Token {
	t := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return t
}

// parseExpression: term ( ("+" | "-") term )*
func (p *Parser) parseExpression() (Node, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}

	for p.peek().Type == TOKEN_PLUS || p.peek().Type == TOKEN_MINUS {
		op := p.advance()
		right, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op.Type, Left: left, Right: right}
	}

	return left, nil
}

// parseTerm: unary ( ("*" | "/") unary )*
func (p *Parser) parseTerm() (Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for p.peek().Type == TOKEN_STAR || p.peek().Type == TOKEN_SLASH {
		op := p.advance()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op.Type, Left: left, Right: right}
	}

	return left, nil
}

// parseUnary: "-" unary | postfix
func (p *Parser) parseUnary() (Node, error) {
	if p.peek().Type == TOKEN_MINUS {
		op := p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: op.Type, Operand: operand}, nil
	}
	return p.parsePostfix()
}

// parsePostfix: primary ("%"? unit?)
func (p *Parser) parsePostfix() (Node, error) {
	node, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	// Check for % postfix
	if p.peek().Type == TOKEN_PERCENT {
		p.advance() // consume '%'
		node = &PercentExpr{Expr: node}
		return node, nil
	}

	// Check for AM/PM postfix on time-producing nodes before unit lookup
	// (avoids "pm" matching picometers instead of PM)
	if p.peek().Type == TOKEN_WORD && isAMPM(p.peek().Literal) {
		if isTimeProducing(node) {
			isPM := strings.EqualFold(p.advance().Literal, "PM")
			node = &AMPMExpr{Expr: node, IsPM: isPM}
		}
	}

	// Check for timezone postfix on time-producing nodes (e.g. "12:00 UTC")
	if p.peek().Type == TOKEN_WORD && IsTimezone(p.peek().Literal) {
		if isTimeProducing(node) {
			tz := p.advance().Literal
			return &TZExpr{Expr: node, TZ: tz, IsInput: true}, nil
		}
	}

	// Check if next token is a WORD that matches a known unit
	if p.peek().Type == TOKEN_WORD {
		u := LookupUnit(p.peek().Literal)
		if u != nil {
			p.advance() // consume the unit token
			return &UnitExpr{Expr: node, Unit: SimpleUnit(*u)}, nil
		}
	}

	return node, nil
}

// parsePrimary: number | varname | "(" expression ")"
func (p *Parser) parsePrimary() (Node, error) {
	tok := p.peek()

	switch tok.Type {
	case TOKEN_NUMBER:
		return p.parseNumber()

	case TOKEN_AT:
		p.advance() // consume @ token
		return parseAtLiteral(tok.Literal)

	case TOKEN_TIME:
		p.advance() // consume time token
		return &TimeLit{Raw: tok.Literal}, nil

	case TOKEN_LPAREN:
		p.advance() // consume '('
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.peek().Type != TOKEN_RPAREN {
			return nil, &EvalError{Msg: "expected ')'"}
		}
		p.advance() // consume ')'
		return expr, nil

	case TOKEN_HASH:
		// #NUMBER → line reference variable
		p.advance() // consume '#'
		if p.peek().Type != TOKEN_NUMBER {
			return nil, &EvalError{Msg: "expected number after #"}
		}
		num := p.advance()
		return &VarRef{Name: "#" + num.Literal}, nil

	case TOKEN_WORD:
		// Check if this is a function call: WORD followed by LPAREN
		if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == TOKEN_LPAREN {
			return p.parseFuncCall()
		}
		return p.parseVarRef()

	default:
		return nil, &EvalError{Msg: "unexpected token: " + tok.Literal}
	}
}

// parseNumber: NUMBER ( "." NUMBER )? ( "/" NUMBER )?
func (p *Parser) parseNumber() (Node, error) {
	intTok := p.advance() // consume integer part

	// Check for 0x, 0b, 0o prefixed literals
	lit := intTok.Literal
	if len(lit) >= 2 && lit[0] == '0' {
		prefix := lit[1]
		if prefix == 'x' || prefix == 'X' || prefix == 'b' || prefix == 'B' || prefix == 'o' || prefix == 'O' {
			var base int
			switch prefix {
			case 'x', 'X':
				base = 16
			case 'b', 'B':
				base = 2
			case 'o', 'O':
				base = 8
			}
			z := new(big.Int)
			if _, ok := z.SetString(lit[2:], base); !ok {
				return nil, &EvalError{Msg: "invalid number: " + lit}
			}
			r := new(big.Rat).SetInt(z)
			return &NumberLit{Value: r}, nil
		}
	}

	// Check for decimal: NUMBER "." NUMBER
	if p.peek().Type == TOKEN_DOT {
		p.advance() // consume '.'
		if p.peek().Type != TOKEN_NUMBER {
			return nil, &EvalError{Msg: "expected digits after decimal point"}
		}
		fracTok := p.advance()
		// Build rational from decimal
		decStr := intTok.Literal + "." + fracTok.Literal
		r := new(big.Rat)
		if _, ok := r.SetString(decStr); !ok {
			return nil, &EvalError{Msg: "invalid number: " + decStr}
		}
		return &NumberLit{Value: r}, nil
	}

	// Check for fraction: NUMBER "/" NUMBER
	// But only if the next token is SLASH and the one after is NUMBER
	// and there's no space suggesting it's division
	if p.peek().Type == TOKEN_SLASH && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == TOKEN_NUMBER {
		// Check if the slash is adjacent to both numbers (no space = fraction literal)
		slashTok := p.tokens[p.pos]
		denomTok := p.tokens[p.pos+1]
		if slashTok.Pos == intTok.Pos+len(intTok.Literal) &&
			denomTok.Pos == slashTok.Pos+1 {
			p.advance() // consume '/'
			p.advance() // consume denominator
			ratStr := intTok.Literal + "/" + denomTok.Literal
			r := new(big.Rat)
			if _, ok := r.SetString(ratStr); !ok {
				return nil, &EvalError{Msg: "invalid fraction: " + ratStr}
			}
			return &NumberLit{Value: r}, nil
		}
	}

	// Plain integer
	r := new(big.Rat)
	r.SetString(intTok.Literal)
	return &NumberLit{Value: r}, nil
}

// parseFuncCall: WORD "(" [expression ("," expression)*] ")"
func (p *Parser) parseFuncCall() (Node, error) {
	name := p.advance().Literal // consume function name
	p.advance()                 // consume '('

	var args []Node
	if p.peek().Type != TOKEN_RPAREN {
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		for p.peek().Type == TOKEN_COMMA {
			p.advance() // consume ','
			arg, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		}
	}

	if p.peek().Type != TOKEN_RPAREN {
		return nil, &EvalError{Msg: "expected ')' in function call"}
	}
	p.advance() // consume ')'
	return &FuncCall{Name: name, Args: args}, nil
}

// parseVarRef: single WORD token as variable name.
func (p *Parser) parseVarRef() (Node, error) {
	if p.peek().Type != TOKEN_WORD {
		return nil, &EvalError{Msg: "expected variable name"}
	}
	return &VarRef{Name: p.advance().Literal}, nil
}

// parseConversion checks for "to" followed by a compound unit spec or timezone.
// "to" is context-sensitive: only treated as a keyword when followed by a known unit or timezone.
func (p *Parser) parseConversion(expr Node) (Node, error) {
	if p.peek().Type != TOKEN_WORD || p.peek().Literal != "to" {
		return expr, nil
	}
	// Look ahead: the token after "to" must be a known unit or timezone
	if p.pos+1 >= len(p.tokens) || p.tokens[p.pos+1].Type != TOKEN_WORD {
		return expr, nil
	}
	nextWord := p.tokens[p.pos+1].Literal
	// Check for timezone conversion
	if IsTimezone(nextWord) {
		p.advance() // consume "to"
		tz := p.advance().Literal
		return &TZExpr{Expr: expr, TZ: tz, IsInput: false}, nil
	}
	// Check for "to unix" — convert time to unix timestamp number
	if nextWord == "unix" {
		p.advance() // consume "to"
		p.advance() // consume "unix"
		return &FuncCall{Name: "__to_unix", Args: []Node{expr}}, nil
	}
	// Check for "to hex/bin/oct" — base conversion
	if nextWord == "hex" {
		p.advance() // consume "to"
		p.advance() // consume "hex"
		return &FuncCall{Name: "__to_hex", Args: []Node{expr}}, nil
	}
	if nextWord == "bin" {
		p.advance() // consume "to"
		p.advance() // consume "bin"
		return &FuncCall{Name: "__to_bin", Args: []Node{expr}}, nil
	}
	if nextWord == "oct" {
		p.advance() // consume "to"
		p.advance() // consume "oct"
		return &FuncCall{Name: "__to_oct", Args: []Node{expr}}, nil
	}
	// Check for unit conversion
	if LookupUnit(nextWord) == nil {
		return expr, nil
	}
	p.advance() // consume "to"
	unit, err := p.parseCompoundUnitSpec()
	if err != nil {
		return nil, err
	}
	return &UnitExpr{Expr: expr, Unit: unit}, nil
}

// isAMPM returns true if s is "AM" or "PM" (case-insensitive).
func isAMPM(s string) bool {
	return strings.EqualFold(s, "AM") || strings.EqualFold(s, "PM")
}

// isTimeProducing returns true if the node produces a time value (for timezone/AM-PM postfix).
func isTimeProducing(node Node) bool {
	switch node.(type) {
	case *TimeLit, *FuncCall, *AMPMExpr:
		return true
	default:
		return false
	}
}

// parseCompoundUnitSpec parses a compound unit like "km/L".
// Grammar: UNIT ("/" UNIT)?
func (p *Parser) parseCompoundUnitSpec() (CompoundUnit, error) {
	if p.peek().Type != TOKEN_WORD {
		return CompoundUnit{}, &EvalError{Msg: "expected unit after 'to'"}
	}
	first := p.advance()
	u := LookupUnit(first.Literal)
	if u == nil {
		return CompoundUnit{}, &EvalError{Msg: "unknown unit: " + first.Literal}
	}
	cu := CompoundUnit{Num: *u, Den: numUnit}

	if p.peek().Type == TOKEN_SLASH {
		p.advance() // consume '/'
		if p.peek().Type != TOKEN_WORD {
			return CompoundUnit{}, &EvalError{Msg: "expected unit after '/'"}
		}
		word := p.advance()
		den := LookupUnit(word.Literal)
		if den == nil {
			return CompoundUnit{}, &EvalError{Msg: "unknown unit: " + word.Literal}
		}
		cu.Den = *den
	}
	return cu, nil
}

// parseAtLiteral desugars an @-prefixed literal into a FuncCall.
// "@2024-01-31" → Date(2024, 1, 31)
// "@2024-01-31T10:30:00" → Date(2024, 1, 31, 10, 30, 0)
// "@2024-01-31 10:30:00" → Date(2024, 1, 31, 10, 30, 0)
// "@2024-01-31 10:30:00 +0530" → Date(2024, 1, 31, 10, 30, 0) - 19800
// "@10:30" → Time(10, 30)
// "@10:30:00" → Time(10, 30, 0)
func parseAtLiteral(lit string) (Node, error) {
	raw := lit[1:] // strip leading @

	if strings.Contains(raw, "-") {
		// Date or datetime, possibly with timezone offset
		// Check for trailing " +NNNN" or " -NNNN" offset
		var offsetSeconds int64
		if len(raw) >= 6 {
			tail := raw[len(raw)-6:]
			if tail[0] == ' ' && (tail[1] == '+' || tail[1] == '-') &&
				isAllDigits(tail[2:6]) {
				hh, _ := strconv.Atoi(tail[2:4])
				mm, _ := strconv.Atoi(tail[4:6])
				offsetSeconds = int64(hh*3600 + mm*60)
				if tail[1] == '-' {
					offsetSeconds = -offsetSeconds
				}
				raw = raw[:len(raw)-6]
			}
		}

		// Split date from optional time (separator is 'T' or ' ')
		var datePart, timePart string
		if idx := strings.IndexByte(raw, 'T'); idx >= 0 {
			datePart = raw[:idx]
			timePart = raw[idx+1:]
		} else if idx := strings.IndexByte(raw, ' '); idx >= 0 {
			datePart = raw[:idx]
			timePart = raw[idx+1:]
		} else {
			datePart = raw
		}

		dateParts := strings.Split(datePart, "-")
		if len(dateParts) != 3 {
			return nil, &EvalError{Msg: "invalid @ literal: " + lit}
		}
		args := []Node{intNode(dateParts[0]), intNode(dateParts[1]), intNode(dateParts[2])}
		if timePart != "" {
			timeParts := strings.Split(timePart, ":")
			if len(timeParts) != 3 {
				return nil, &EvalError{Msg: "invalid @ literal: " + lit}
			}
			args = append(args, intNode(timeParts[0]), intNode(timeParts[1]), intNode(timeParts[2]))
		}

		var node Node = &FuncCall{Name: "date", Args: args}
		// Adjust for timezone offset: the components are in the given offset,
		// but Date() treats them as UTC, so subtract the offset.
		if offsetSeconds != 0 {
			offsetNode := &UnitExpr{
				Expr: &NumberLit{Value: new(big.Rat).SetInt64(offsetSeconds)},
				Unit: SimpleUnit(*SecondsUnit()),
			}
			node = &BinaryExpr{Op: TOKEN_MINUS, Left: node, Right: offsetNode}
		}
		return node, nil
	}

	if strings.Contains(raw, ":") {
		// Time
		timeParts := strings.Split(raw, ":")
		if len(timeParts) < 2 || len(timeParts) > 3 {
			return nil, &EvalError{Msg: "invalid @ literal: " + lit}
		}
		args := []Node{intNode(timeParts[0]), intNode(timeParts[1])}
		if len(timeParts) == 3 {
			args = append(args, intNode(timeParts[2]))
		}
		return &FuncCall{Name: "time", Args: args}, nil
	}

	// Fallback: plain digits → unix timestamp
	r := new(big.Rat)
	r.SetString(raw)
	return &FuncCall{Name: "unix", Args: []Node{&NumberLit{Value: r}}}, nil
}

func intNode(s string) Node {
	n, _ := strconv.Atoi(s)
	return &NumberLit{Value: new(big.Rat).SetInt64(int64(n))}
}

func isAllDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return len(s) > 0
}

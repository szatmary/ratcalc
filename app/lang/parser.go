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

	// Detect assignment: find the LAST "=" that isn't inside parens
	lastEq := findLastEquals(tokens)
	if lastEq >= 0 {
		return p.parseAssignment(lastEq)
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

// findLastEquals finds the index of the last EQUALS token not inside parentheses.
// Returns -1 if no such token exists.
func findLastEquals(tokens []Token) int {
	depth := 0
	last := -1
	for i, t := range tokens {
		switch t.Type {
		case TOKEN_LPAREN:
			depth++
		case TOKEN_RPAREN:
			depth--
		case TOKEN_EQUALS:
			if depth == 0 {
				last = i
			}
		}
	}
	if last < 0 {
		return -1
	}
	// The LHS must be non-empty and consist only of WORD tokens
	// to be a valid assignment
	lhs := tokens[:last]
	if len(lhs) == 0 {
		return -1
	}
	for _, t := range lhs {
		if t.Type != TOKEN_WORD {
			return -1
		}
	}
	return last
}

func (p *Parser) parseAssignment(eqIdx int) (Node, error) {
	// Build variable name from LHS WORD tokens
	var parts []string
	for i := 0; i < eqIdx; i++ {
		parts = append(parts, p.tokens[i].Literal)
	}
	name := strings.Join(parts, " ")

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

// parsePostfix: primary unit?
func (p *Parser) parsePostfix() (Node, error) {
	node, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	// Check if next token is a WORD that matches a known unit
	if p.peek().Type == TOKEN_WORD {
		u := LookupUnit(p.peek().Literal)
		if u != nil {
			p.advance() // consume the unit token
			return &UnitExpr{Expr: node, Unit: SimpleUnit(u)}, nil
		}
	}

	// Check for AM/PM postfix on time-producing nodes (e.g. "3:30 PM")
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

// parseVarRef: WORD ( WORD )* — greedy, multi-word variable name.
// Stops when the next WORD is a known unit (since that's handled by postfix).
// But only stops for units if the current node is a number context — since
// we're in varRef, all consecutive WORDs are the variable name.
func (p *Parser) parseVarRef() (Node, error) {
	var parts []string
	for p.peek().Type == TOKEN_WORD {
		// Don't consume a word that is a unit name if it would leave us
		// with at least one word already collected. This prevents
		// "price in kg" from treating "kg" as part of the variable name.
		// However, a standalone unit name IS a valid variable name.
		if len(parts) > 0 {
			// Look ahead: if this word is a unit and it's the last WORD
			// before a non-WORD token, it might be a unit. But in varRef
			// context (not after a number), it's part of the variable name.
			// We consume it.
		}
		parts = append(parts, p.advance().Literal)
	}
	if len(parts) == 0 {
		return nil, &EvalError{Msg: "expected variable name"}
	}
	name := strings.Join(parts, " ")
	return &VarRef{Name: name}, nil
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

// parseCompoundUnitSpec parses a compound unit like "km/L" or "m*s/kg".
// Grammar: UNIT (("/" | "*") UNIT)*
func (p *Parser) parseCompoundUnitSpec() (*CompoundUnit, error) {
	if p.peek().Type != TOKEN_WORD {
		return nil, &EvalError{Msg: "expected unit after 'to'"}
	}
	first := p.advance()
	u := LookupUnit(first.Literal)
	if u == nil {
		return nil, &EvalError{Msg: "unknown unit: " + first.Literal}
	}
	cu := &CompoundUnit{Num: []*Unit{u}}

	for p.peek().Type == TOKEN_SLASH || p.peek().Type == TOKEN_STAR {
		op := p.advance()
		if p.peek().Type != TOKEN_WORD {
			return nil, &EvalError{Msg: "expected unit after '" + op.Literal + "'"}
		}
		word := p.advance()
		next := LookupUnit(word.Literal)
		if next == nil {
			return nil, &EvalError{Msg: "unknown unit: " + word.Literal}
		}
		if op.Type == TOKEN_SLASH {
			cu.Den = append(cu.Den, next)
		} else {
			cu.Num = append(cu.Num, next)
		}
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
				Unit: SimpleUnit(SecondsUnit()),
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

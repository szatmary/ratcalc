package lang

import "math/big"

// Node is the interface all AST nodes implement.
type Node interface {
	nodeTag()
}

// NumberLit represents a number literal (integer or decimal).
type NumberLit struct {
	Value *big.Rat
}

// VarRef represents a variable reference (possibly multi-word).
type VarRef struct {
	Name string
}

// BinaryExpr represents a binary operation.
type BinaryExpr struct {
	Op    TokenType // TOKEN_PLUS, TOKEN_MINUS, TOKEN_STAR, TOKEN_SLASH, TOKEN_STARSTAR, TOKEN_AMP, TOKEN_PIPE, TOKEN_CARET, TOKEN_LSHIFT, TOKEN_RSHIFT
	Left  Node
	Right Node
}

// UnaryExpr represents a unary operation (negation or bitwise NOT).
type UnaryExpr struct {
	Op      TokenType // TOKEN_MINUS, TOKEN_TILDE
	Operand Node
}

// UnitExpr wraps an expression with a unit annotation.
type UnitExpr struct {
	Expr Node
	Unit CompoundUnit
}

// Assignment represents name = expression.
type Assignment struct {
	Name string
	Expr Node
}

// FuncCall represents a function call like Now(), Date(), Time(), or __unix(expr).
type FuncCall struct {
	Name string
	Args []Node
}

// TimeLit represents a time-of-day literal like "12:00" or "14:30:00".
type TimeLit struct {
	Raw string
}

// TZExpr wraps an expression with a timezone annotation or conversion.
// IsInput=true means the time was entered in this timezone (postfix like "12:00 UTC").
// IsInput=false means convert display to this timezone ("to PST").
type TZExpr struct {
	Expr    Node
	TZ      string
	IsInput bool
}

// PercentExpr wraps an expression with a % suffix, dividing by 100.
type PercentExpr struct {
	Expr Node
}

// FactorialExpr wraps an expression with a ! suffix (factorial).
type FactorialExpr struct {
	Expr Node
}

func (*NumberLit) nodeTag()   {}
func (*VarRef) nodeTag()      {}
func (*BinaryExpr) nodeTag()  {}
func (*UnaryExpr) nodeTag()   {}
func (*UnitExpr) nodeTag()    {}
func (*Assignment) nodeTag()  {}
func (*FuncCall) nodeTag()    {}
func (*TimeLit) nodeTag()     {}
func (*TZExpr) nodeTag()      {}
func (*AMPMExpr) nodeTag()    {}
func (*PercentExpr) nodeTag()   {}
func (*FactorialExpr) nodeTag() {}

// AMPMExpr wraps a time-producing expression with an AM/PM modifier.
type AMPMExpr struct {
	Expr Node
	IsPM bool
}

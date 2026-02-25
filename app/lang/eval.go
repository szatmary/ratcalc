package lang

import (
	"fmt"
	"math/big"
	"time"
)

// Env is the variable environment mapping names to values.
type Env map[string]Value

// Eval evaluates an AST node in the given environment.
func Eval(node Node, env Env) (Value, error) {
	if node == nil {
		return Value{}, &EvalError{Msg: "empty expression"}
	}

	switch n := node.(type) {
	case *NumberLit:
		return Value{Rat: new(big.Rat).Set(n.Value)}, nil

	case *RatioLit:
		if n.Denom.Sign() == 0 {
			return Value{}, &EvalError{Msg: "division by zero in ratio"}
		}
		r := new(big.Rat).Quo(n.Num, n.Denom)
		return Value{Rat: r}, nil

	case *VarRef:
		v, ok := env[n.Name]
		if !ok {
			// Try looking up as a unit — bare unit word implies 1
			if u := LookupUnit(n.Name); u != nil {
				return Value{Rat: new(big.Rat).SetInt64(1), Unit: SimpleUnit(u)}, nil
			}
			return Value{}, &EvalError{Msg: "undefined variable: " + n.Name}
		}
		return v, nil

	case *BinaryExpr:
		left, err := Eval(n.Left, env)
		if err != nil {
			return Value{}, err
		}
		right, err := Eval(n.Right, env)
		if err != nil {
			return Value{}, err
		}
		switch n.Op {
		case TOKEN_PLUS:
			return valAdd(left, right)
		case TOKEN_MINUS:
			return valSub(left, right)
		case TOKEN_STAR:
			return valMul(left, right)
		case TOKEN_SLASH:
			return valDiv(left, right)
		default:
			return Value{}, &EvalError{Msg: "unknown operator"}
		}

	case *UnaryExpr:
		operand, err := Eval(n.Operand, env)
		if err != nil {
			return Value{}, err
		}
		if n.Op == TOKEN_MINUS {
			return valNeg(operand), nil
		}
		return Value{}, &EvalError{Msg: "unknown unary operator"}

	case *UnitExpr:
		val, err := Eval(n.Expr, env)
		if err != nil {
			return Value{}, err
		}
		if val.Unit != nil {
			// Already has a unit — convert if compatible
			if !val.Unit.Compatible(n.Unit) {
				return Value{}, &EvalError{Msg: "cannot convert " + val.Unit.String() + " to " + n.Unit.String()}
			}
			factor := compoundConversionFactor(val.Unit, n.Unit)
			converted := new(big.Rat).Mul(val.Rat, factor)
			return Value{Rat: converted, Unit: n.Unit}, nil
		}
		val.Unit = n.Unit
		return val, nil

	case *Assignment:
		val, err := Eval(n.Expr, env)
		if err != nil {
			return Value{}, err
		}
		env[n.Name] = val
		return val, nil

	case *FuncCall:
		return evalFuncCall(n, env)

	case *TimeLit:
		return evalTimeLit(n.Raw)

	case *TZExpr:
		return evalTZExpr(n, env)

	case *AMPMExpr:
		return evalAMPM(n, env)

	default:
		return Value{}, &EvalError{Msg: "unknown node type"}
	}
}

// ParseLine lexes and parses a single line into an AST node without evaluating.
func ParseLine(line string) (Node, error) {
	tokens := Lex(line)
	allEOF := true
	for _, t := range tokens {
		if t.Type != TOKEN_EOF {
			allEOF = false
			break
		}
	}
	if allEOF {
		return nil, nil
	}
	return Parse(tokens)
}

func evalTimeLit(raw string) (Value, error) {
	// Parse HH:MM or HH:MM:SS
	var h, m, s int
	var err error
	if len(raw) > 5 {
		// HH:MM:SS
		_, err = fmt.Sscanf(raw, "%d:%d:%d", &h, &m, &s)
	} else {
		// HH:MM (or H:MM)
		_, err = fmt.Sscanf(raw, "%d:%d", &h, &m)
	}
	if err != nil {
		return Value{}, &EvalError{Msg: "invalid time: " + raw}
	}
	if h < 0 || h > 23 || m < 0 || m > 59 || s < 0 || s > 59 {
		return Value{}, &EvalError{Msg: "invalid time: " + raw}
	}
	// Get today's date in UTC, set the time
	now := time.Now().UTC()
	t := time.Date(now.Year(), now.Month(), now.Day(), h, m, s, 0, time.UTC)
	r := new(big.Rat).SetInt64(t.Unix())
	return Value{Rat: r, IsTime: true}, nil
}

func evalAMPM(n *AMPMExpr, env Env) (Value, error) {
	val, err := Eval(n.Expr, env)
	if err != nil {
		return Value{}, err
	}
	if !val.IsTime {
		return Value{}, &EvalError{Msg: "AM/PM can only be applied to time values"}
	}
	// Extract the hour from the UTC time
	unix := val.Rat.Num().Int64() / val.Rat.Denom().Int64()
	t := time.Unix(unix, 0).UTC()
	h := t.Hour()

	if n.IsPM {
		if h < 12 {
			// e.g. 3:30 PM → add 12 hours
			val.Rat = new(big.Rat).Add(val.Rat, new(big.Rat).SetInt64(12*3600))
		}
		// h == 12 (12:00 PM = noon) → no change
		// h > 12 shouldn't happen for valid 12-hour input
	} else {
		// AM
		if h == 12 {
			// 12:00 AM = midnight → subtract 12 hours
			val.Rat = new(big.Rat).Sub(val.Rat, new(big.Rat).SetInt64(12*3600))
		}
		// h 1-11 AM → no change
	}
	return val, nil
}

func evalTZExpr(n *TZExpr, env Env) (Value, error) {
	val, err := Eval(n.Expr, env)
	if err != nil {
		return Value{}, err
	}
	if !val.IsTime {
		return Value{}, &EvalError{Msg: "timezone can only be applied to time values"}
	}
	loc := LookupTimezone(n.TZ)
	if loc == nil {
		return Value{}, &EvalError{Msg: "unknown timezone: " + n.TZ}
	}
	if n.IsInput {
		// The time literal was specified in this timezone.
		// Subtract the UTC offset so internal value becomes correct UTC.
		_, offset := time.Unix(val.Rat.Num().Int64()/val.Rat.Denom().Int64(), 0).In(loc).Zone()
		adjustment := new(big.Rat).SetInt64(int64(offset))
		val.Rat = new(big.Rat).Sub(val.Rat, adjustment)
	}
	val.TZ = loc
	return val, nil
}

func evalFuncCall(n *FuncCall, env Env) (Value, error) {
	switch n.Name {
	case "Now":
		if len(n.Args) != 0 {
			return Value{}, &EvalError{Msg: "Now() takes no arguments"}
		}
		r := new(big.Rat).SetInt64(time.Now().Unix())
		return Value{Rat: r, IsTime: true}, nil

	case "Date":
		if len(n.Args) != 3 && len(n.Args) != 6 {
			return Value{}, &EvalError{Msg: "Date() takes 3 or 6 arguments"}
		}
		vals := make([]int, len(n.Args))
		for i, arg := range n.Args {
			v, err := Eval(arg, env)
			if err != nil {
				return Value{}, err
			}
			if !v.Rat.IsInt() {
				return Value{}, &EvalError{Msg: "Date() arguments must be integers"}
			}
			vals[i] = int(v.Rat.Num().Int64())
		}
		var t time.Time
		if len(vals) == 3 {
			t = time.Date(vals[0], time.Month(vals[1]), vals[2], 0, 0, 0, 0, time.UTC)
		} else {
			t = time.Date(vals[0], time.Month(vals[1]), vals[2], vals[3], vals[4], vals[5], 0, time.UTC)
		}
		r := new(big.Rat).SetInt64(t.Unix())
		return Value{Rat: r, IsTime: true}, nil

	case "Time":
		if len(n.Args) != 2 && len(n.Args) != 3 {
			return Value{}, &EvalError{Msg: "Time() takes 2 or 3 arguments"}
		}
		vals := make([]int, len(n.Args))
		for i, arg := range n.Args {
			v, err := Eval(arg, env)
			if err != nil {
				return Value{}, err
			}
			if !v.Rat.IsInt() {
				return Value{}, &EvalError{Msg: "Time() arguments must be integers"}
			}
			vals[i] = int(v.Rat.Num().Int64())
		}
		h, m := vals[0], vals[1]
		s := 0
		if len(vals) == 3 {
			s = vals[2]
		}
		if h < 0 || h > 23 || m < 0 || m > 59 || s < 0 || s > 59 {
			return Value{}, &EvalError{Msg: "invalid time"}
		}
		now := time.Now().UTC()
		tt := time.Date(now.Year(), now.Month(), now.Day(), h, m, s, 0, time.UTC)
		r := new(big.Rat).SetInt64(tt.Unix())
		return Value{Rat: r, IsTime: true}, nil

	case "__to_unix":
		if len(n.Args) != 1 {
			return Value{}, &EvalError{Msg: "to unix requires a value"}
		}
		val, err := Eval(n.Args[0], env)
		if err != nil {
			return Value{}, err
		}
		if !val.IsTime {
			return Value{}, &EvalError{Msg: "to unix requires a time value"}
		}
		return Value{Rat: new(big.Rat).Set(val.Rat), Base: 10}, nil

	case "__to_hex", "__to_bin", "__to_oct":
		if len(n.Args) != 1 {
			return Value{}, &EvalError{Msg: "to " + n.Name[5:] + " requires a value"}
		}
		val, err := Eval(n.Args[0], env)
		if err != nil {
			return Value{}, err
		}
		if !val.Rat.IsInt() {
			return Value{}, &EvalError{Msg: "to " + n.Name[5:] + " requires an integer"}
		}
		var base int
		switch n.Name {
		case "__to_hex":
			base = 16
		case "__to_bin":
			base = 2
		case "__to_oct":
			base = 8
		}
		return Value{Rat: new(big.Rat).Set(val.Rat), Unit: val.Unit, Base: base}, nil

	case "Unix":
		if len(n.Args) != 1 {
			return Value{}, &EvalError{Msg: "Unix() takes 1 argument"}
		}
		val, err := Eval(n.Args[0], env)
		if err != nil {
			return Value{}, err
		}
		if val.Unit != nil {
			return Value{}, &EvalError{Msg: "Unix() value must be dimensionless"}
		}
		r := autoDetectUnixPrecision(val.Rat)
		return Value{Rat: r, IsTime: true}, nil

	default:
		return Value{}, &EvalError{Msg: "unknown function: " + n.Name}
	}
}

// autoDetectUnixPrecision converts a unix timestamp to seconds, auto-detecting
// if the input is in seconds, milliseconds, microseconds, or nanoseconds.
func autoDetectUnixPrecision(r *big.Rat) *big.Rat {
	// Get the integer value for threshold comparison
	v := new(big.Rat).Set(r)
	if v.Sign() < 0 {
		v.Neg(v)
	}

	threshMs := new(big.Rat).SetInt64(1e12)
	threshUs := new(big.Rat).SetInt64(1e15)
	threshNs := new(big.Rat).SetInt64(1e18)

	result := new(big.Rat).Set(r)
	if v.Cmp(threshMs) < 0 {
		// seconds — already correct
		return result
	} else if v.Cmp(threshUs) < 0 {
		// milliseconds → divide by 1000
		return result.Quo(result, new(big.Rat).SetInt64(1000))
	} else if v.Cmp(threshNs) < 0 {
		// microseconds → divide by 1e6
		return result.Quo(result, new(big.Rat).SetInt64(1e6))
	}
	// nanoseconds → divide by 1e9
	return result.Quo(result, new(big.Rat).SetInt64(1e9))
}

// EvalLine lexes, parses, and evaluates a single line.
func EvalLine(line string, env Env) (Value, error) {
	tokens := Lex(line)

	// Check if the line is empty (only EOF)
	allEOF := true
	for _, t := range tokens {
		if t.Type != TOKEN_EOF {
			allEOF = false
			break
		}
	}
	if allEOF {
		return Value{}, &EvalError{Msg: ""}
	}

	node, err := Parse(tokens)
	if err != nil {
		return Value{}, err
	}
	if node == nil {
		return Value{}, &EvalError{Msg: ""}
	}
	return Eval(node, env)
}

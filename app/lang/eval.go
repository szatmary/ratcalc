package lang

import (
	"fmt"
	"math"
	"math/big"
	"time"
)

var (
	piRat = new(big.Rat).SetFloat64(math.Pi)
	eRat  = new(big.Rat).SetFloat64(math.E)
	cRat = new(big.Rat).SetInt64(299792458) // speed of light in m/s
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
			// Built-in constants
			switch n.Name {
			case "pi":
				return Value{Rat: new(big.Rat).Set(piRat), Base: 10}, nil
			case "e":
				return Value{Rat: new(big.Rat).Set(eRat), Base: 10}, nil
			case "c":
				return Value{Rat: new(big.Rat).Set(cRat), Base: 10}, nil
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

	case *PercentExpr:
		val, err := Eval(n.Expr, env)
		if err != nil {
			return Value{}, err
		}
		hundred := new(big.Rat).SetInt64(100)
		r := new(big.Rat).Quo(val.Rat, hundred)
		return Value{Rat: r, Base: 10}, nil

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
			// Offset-based conversion (temperature)
			if val.Unit.HasOffset() || n.Unit.HasOffset() {
				// Offset units only allowed as simple units (1 num, 0 den)
				if len(val.Unit.Num) != 1 || len(val.Unit.Den) != 0 ||
					len(n.Unit.Num) != 1 || len(n.Unit.Den) != 0 {
					return Value{}, &EvalError{Msg: "temperature units cannot be used in compound units"}
				}
				from := val.Unit.Num[0]
				to := n.Unit.Num[0]
				v := new(big.Rat).Set(val.Rat)
				// Convert to base (kelvin): kelvin = (val + from.PreOffset) * from.ToBase
				if from.PreOffset != nil {
					v.Add(v, from.PreOffset)
				}
				v.Mul(v, from.ToBase)
				// Convert from base to target: result = kelvin / to.ToBase - to.PreOffset
				v.Quo(v, to.ToBase)
				if to.PreOffset != nil {
					v.Sub(v, to.PreOffset)
				}
				return Value{Rat: v, Unit: n.Unit, Base: 10}, nil
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

func evalMathFunc1(n *FuncCall, env Env, fn func(float64) float64) (Value, error) {
	if len(n.Args) != 1 {
		return Value{}, &EvalError{Msg: n.Name + "() takes 1 argument"}
	}
	val, err := Eval(n.Args[0], env)
	if err != nil {
		return Value{}, err
	}
	if val.Unit != nil {
		return Value{}, &EvalError{Msg: n.Name + "() requires a dimensionless value"}
	}
	if val.IsTime {
		return Value{}, &EvalError{Msg: n.Name + "() cannot operate on time values"}
	}
	f, _ := val.Rat.Float64()
	result := fn(f)
	r := new(big.Rat).SetFloat64(result)
	if r == nil {
		return Value{}, &EvalError{Msg: n.Name + "(): result out of range"}
	}
	return Value{Rat: r, Base: 10}, nil
}

func evalMathFunc2(n *FuncCall, env Env, fn func(float64, float64) float64) (Value, error) {
	if len(n.Args) != 2 {
		return Value{}, &EvalError{Msg: n.Name + "() takes 2 arguments"}
	}
	a, err := Eval(n.Args[0], env)
	if err != nil {
		return Value{}, err
	}
	b, err := Eval(n.Args[1], env)
	if err != nil {
		return Value{}, err
	}
	if a.Unit != nil || a.IsTime {
		return Value{}, &EvalError{Msg: n.Name + "() requires dimensionless values"}
	}
	if b.Unit != nil || b.IsTime {
		return Value{}, &EvalError{Msg: n.Name + "() requires dimensionless values"}
	}
	af, _ := a.Rat.Float64()
	bf, _ := b.Rat.Float64()
	result := fn(af, bf)
	r := new(big.Rat).SetFloat64(result)
	if r == nil {
		return Value{}, &EvalError{Msg: n.Name + "(): result out of range"}
	}
	return Value{Rat: r, Base: 10}, nil
}

func evalFinanceFunc3(n *FuncCall, env Env, fn func(float64, float64, float64) float64) (Value, error) {
	if len(n.Args) != 3 {
		return Value{}, &EvalError{Msg: n.Name + "() takes 3 arguments"}
	}
	vals := make([]float64, 3)
	for i, arg := range n.Args {
		v, err := Eval(arg, env)
		if err != nil {
			return Value{}, err
		}
		if v.Unit != nil || v.IsTime {
			return Value{}, &EvalError{Msg: n.Name + "() requires dimensionless values"}
		}
		vals[i], _ = v.Rat.Float64()
	}
	result := fn(vals[0], vals[1], vals[2])
	r := new(big.Rat).SetFloat64(result)
	if r == nil {
		return Value{}, &EvalError{Msg: n.Name + "(): result out of range"}
	}
	return Value{Rat: r, Base: 10}, nil
}

func evalTimeExtract(n *FuncCall, env Env, extract func(time.Time) int) (Value, error) {
	if len(n.Args) != 1 {
		return Value{}, &EvalError{Msg: n.Name + "() takes 1 argument"}
	}
	val, err := Eval(n.Args[0], env)
	if err != nil {
		return Value{}, err
	}
	if !val.IsTime {
		return Value{}, &EvalError{Msg: n.Name + "() requires a time value"}
	}
	unix := val.Rat.Num().Int64() / val.Rat.Denom().Int64()
	loc := time.UTC
	if val.TZ != nil {
		loc = val.TZ
	}
	t := time.Unix(unix, 0).In(loc)
	return Value{Rat: new(big.Rat).SetInt64(int64(extract(t)))}, nil
}

// ratFloor returns floor(x) as an integer-valued *big.Rat.
func ratFloor(x *big.Rat) *big.Rat {
	// Quo truncates toward zero
	q := new(big.Int).Quo(x.Num(), x.Denom())
	// If negative and there's a remainder, subtract 1
	if x.Sign() < 0 {
		rem := new(big.Int).Rem(x.Num(), x.Denom())
		if rem.Sign() != 0 {
			q.Sub(q, big.NewInt(1))
		}
	}
	return new(big.Rat).SetInt(q)
}

// ratCeil returns ceil(x) as an integer-valued *big.Rat.
func ratCeil(x *big.Rat) *big.Rat {
	neg := new(big.Rat).Neg(x)
	return new(big.Rat).Neg(ratFloor(neg))
}

// ratRound returns round(x) with half away from zero.
func ratRound(x *big.Rat) *big.Rat {
	half := new(big.Rat).SetFrac64(1, 2)
	if x.Sign() >= 0 {
		return ratFloor(new(big.Rat).Add(x, half))
	}
	return ratCeil(new(big.Rat).Sub(x, half))
}

func evalRatFunc1(n *FuncCall, env Env, fn func(*big.Rat) *big.Rat) (Value, error) {
	if len(n.Args) != 1 {
		return Value{}, &EvalError{Msg: n.Name + "() takes 1 argument"}
	}
	val, err := Eval(n.Args[0], env)
	if err != nil {
		return Value{}, err
	}
	if val.Unit != nil {
		return Value{}, &EvalError{Msg: n.Name + "() requires a dimensionless value"}
	}
	if val.IsTime {
		return Value{}, &EvalError{Msg: n.Name + "() cannot operate on time values"}
	}
	return Value{Rat: fn(val.Rat), Base: 10}, nil
}

func evalRatFunc2(n *FuncCall, env Env, fn func(*big.Rat, *big.Rat) *big.Rat) (Value, error) {
	if len(n.Args) != 2 {
		return Value{}, &EvalError{Msg: n.Name + "() takes 2 arguments"}
	}
	a, err := Eval(n.Args[0], env)
	if err != nil {
		return Value{}, err
	}
	b, err := Eval(n.Args[1], env)
	if err != nil {
		return Value{}, err
	}
	if a.Unit != nil || a.IsTime {
		return Value{}, &EvalError{Msg: n.Name + "() requires dimensionless values"}
	}
	if b.Unit != nil || b.IsTime {
		return Value{}, &EvalError{Msg: n.Name + "() requires dimensionless values"}
	}
	return Value{Rat: fn(a.Rat, b.Rat), Base: 10}, nil
}

func evalPow(n *FuncCall, env Env) (Value, error) {
	if len(n.Args) != 2 {
		return Value{}, &EvalError{Msg: "pow() takes 2 arguments"}
	}
	base, err := Eval(n.Args[0], env)
	if err != nil {
		return Value{}, err
	}
	exp, err := Eval(n.Args[1], env)
	if err != nil {
		return Value{}, err
	}
	if base.Unit != nil || base.IsTime {
		return Value{}, &EvalError{Msg: "pow() requires dimensionless values"}
	}
	if exp.Unit != nil || exp.IsTime {
		return Value{}, &EvalError{Msg: "pow() requires dimensionless values"}
	}
	// If exponent is integer, use exact rational arithmetic
	if exp.Rat.IsInt() {
		e := exp.Rat.Num().Int64()
		neg := e < 0
		if neg {
			e = -e
		}
		num := new(big.Int).Set(base.Rat.Num())
		den := new(big.Int).Set(base.Rat.Denom())
		num.Exp(num, big.NewInt(e), nil)
		den.Exp(den, big.NewInt(e), nil)
		r := new(big.Rat).SetFrac(num, den)
		if neg {
			if r.Sign() == 0 {
				return Value{}, &EvalError{Msg: "pow(): division by zero"}
			}
			r.Inv(r)
		}
		return Value{Rat: r, Base: 10}, nil
	}
	// Fractional exponent: fall back to float64
	return evalMathFunc2(n, env, math.Pow)
}

func evalFuncCall(n *FuncCall, env Env) (Value, error) {
	switch n.Name {
	case "now":
		if len(n.Args) != 0 {
			return Value{}, &EvalError{Msg: "now() takes no arguments"}
		}
		r := new(big.Rat).SetInt64(time.Now().Unix())
		return Value{Rat: r, IsTime: true}, nil

	case "date":
		if len(n.Args) != 3 && len(n.Args) != 6 {
			return Value{}, &EvalError{Msg: "date() takes 3 or 6 arguments"}
		}
		vals := make([]int, len(n.Args))
		for i, arg := range n.Args {
			v, err := Eval(arg, env)
			if err != nil {
				return Value{}, err
			}
			if !v.Rat.IsInt() {
				return Value{}, &EvalError{Msg: "date() arguments must be integers"}
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

	case "time":
		if len(n.Args) != 2 && len(n.Args) != 3 {
			return Value{}, &EvalError{Msg: "time() takes 2 or 3 arguments"}
		}
		vals := make([]int, len(n.Args))
		for i, arg := range n.Args {
			v, err := Eval(arg, env)
			if err != nil {
				return Value{}, err
			}
			if !v.Rat.IsInt() {
				return Value{}, &EvalError{Msg: "time() arguments must be integers"}
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

	case "unix":
		if len(n.Args) != 1 {
			return Value{}, &EvalError{Msg: "unix() takes 1 argument"}
		}
		val, err := Eval(n.Args[0], env)
		if err != nil {
			return Value{}, err
		}
		if val.Unit != nil {
			return Value{}, &EvalError{Msg: "unix() value must be dimensionless"}
		}
		r := autoDetectUnixPrecision(val.Rat)
		return Value{Rat: r, IsTime: true}, nil

	case "sin":
		return evalMathFunc1(n, env, math.Sin)
	case "cos":
		return evalMathFunc1(n, env, math.Cos)
	case "tan":
		return evalMathFunc1(n, env, math.Tan)
	case "asin":
		return evalMathFunc1(n, env, math.Asin)
	case "acos":
		return evalMathFunc1(n, env, math.Acos)
	case "atan":
		return evalMathFunc1(n, env, math.Atan)
	case "sqrt":
		return evalMathFunc1(n, env, math.Sqrt)
	case "abs":
		return evalRatFunc1(n, env, func(x *big.Rat) *big.Rat { return new(big.Rat).Abs(x) })
	case "log":
		return evalMathFunc1(n, env, math.Log10)
	case "ln":
		return evalMathFunc1(n, env, math.Log)
	case "log2":
		return evalMathFunc1(n, env, math.Log2)
	case "ceil":
		return evalRatFunc1(n, env, ratCeil)
	case "floor":
		return evalRatFunc1(n, env, ratFloor)
	case "round":
		return evalRatFunc1(n, env, ratRound)

	case "pow":
		return evalPow(n, env)
	case "mod":
		return evalRatFunc2(n, env, func(a, b *big.Rat) *big.Rat {
			// mod(a, b) = a - floor(a/b) * b
			q := new(big.Rat).Quo(a, b)
			f := ratFloor(q)
			return new(big.Rat).Sub(a, new(big.Rat).Mul(f, b))
		})
	case "atan2":
		return evalMathFunc2(n, env, math.Atan2)
	case "min":
		return evalRatFunc2(n, env, func(a, b *big.Rat) *big.Rat {
			if a.Cmp(b) <= 0 {
				return new(big.Rat).Set(a)
			}
			return new(big.Rat).Set(b)
		})
	case "max":
		return evalRatFunc2(n, env, func(a, b *big.Rat) *big.Rat {
			if a.Cmp(b) >= 0 {
				return new(big.Rat).Set(a)
			}
			return new(big.Rat).Set(b)
		})

	case "fv":
		return evalFinanceFunc3(n, env, func(rate, nf, pmt float64) float64 {
			// FV = pmt * ((1+rate)^n - 1) / rate
			return pmt * (math.Pow(1+rate, nf) - 1) / rate
		})
	case "pv":
		return evalFinanceFunc3(n, env, func(rate, nf, pmt float64) float64 {
			// PV = pmt * (1 - (1+rate)^(-n)) / rate
			return pmt * (1 - math.Pow(1+rate, -nf)) / rate
		})

	case "year":
		return evalTimeExtract(n, env, func(t time.Time) int { return t.Year() })
	case "month":
		return evalTimeExtract(n, env, func(t time.Time) int { return int(t.Month()) })
	case "day":
		return evalTimeExtract(n, env, func(t time.Time) int { return t.Day() })
	case "hour":
		return evalTimeExtract(n, env, func(t time.Time) int { return t.Hour() })
	case "minute":
		return evalTimeExtract(n, env, func(t time.Time) int { return t.Minute() })
	case "second":
		return evalTimeExtract(n, env, func(t time.Time) int { return t.Second() })

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

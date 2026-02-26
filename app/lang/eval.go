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
	cRat  = new(big.Rat).SetInt64(299792458) // speed of light in m/s
)

// Env is the variable environment mapping names to values.
type Env map[string]Value

// tsVal builds a timestamp Value from a rational (unix seconds).
func tsVal(r *big.Rat) Value {
	return Value{Num: CatVal{Rat: *new(big.Rat).Set(r), Unit: tsUnit}, Den: oneCatVal()}
}

// Eval evaluates an AST node in the given environment.
func Eval(node Node, env Env) (Value, error) {
	if node == nil {
		return Value{}, &EvalError{Msg: "empty expression"}
	}

	switch n := node.(type) {
	case *NumberLit:
		return dimless(n.Value), nil

	case *RatioLit:
		if n.Denom.Sign() == 0 {
			return Value{}, &EvalError{Msg: "division by zero in ratio"}
		}
		r := new(big.Rat).Quo(n.Num, n.Denom)
		return dimless(r), nil

	case *VarRef:
		v, ok := env[n.Name]
		if !ok {
			// Try looking up as a unit — bare unit word implies 1
			if u := LookupUnit(n.Name); u != nil {
				var numRat big.Rat
				numRat.Set(&u.ToBase)
				return Value{Num: CatVal{Rat: numRat, Unit: u}, Den: oneCatVal()}, nil
			}
			// Built-in constants
			switch n.Name {
			case "pi":
				v := dimless(new(big.Rat).Set(piRat))
				v.Display = 10
				return v, nil
			case "e":
				v := dimless(new(big.Rat).Set(eRat))
				v.Display = 10
				return v, nil
			case "c":
				v := dimless(new(big.Rat).Set(cRat))
				v.Display = 10
				return v, nil
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
		eff := val.effectiveRat()
		r := new(big.Rat).Quo(eff, hundred)
		v := dimless(r)
		v.Display = 10
		return v, nil

	case *UnitExpr:
		val, err := Eval(n.Expr, env)
		if err != nil {
			return Value{}, err
		}
		valCU := val.CompoundUnit()
		if !valCU.IsEmpty() {
			// Already has a unit — convert if compatible
			if !valCU.Compatible(n.Unit) {
				return Value{}, &EvalError{Msg: "cannot convert " + valCU.String() + " to " + n.Unit.String()}
			}
			// Offset-based conversion (temperature) — values stored in display units
			if valCU.HasOffset() || n.Unit.HasOffset() {
				// Offset units only allowed as simple units
				if val.Den.Unit != numUnit || n.Unit.Den != numUnit {
					return Value{}, &EvalError{Msg: "temperature units cannot be used in compound units"}
				}
				from := val.Num.Unit
				to := n.Unit.Num
				eff := val.effectiveRat()
				v := new(big.Rat).Set(eff)
				// Convert to base (kelvin): kelvin = (val + from.PreOffset) * from.ToBase
				v.Add(v, &from.PreOffset)
				v.Mul(v, &from.ToBase)
				// Convert from base to target: result = kelvin / to.ToBase - to.PreOffset
				v.Quo(v, &to.ToBase)
				v.Sub(v, &to.PreOffset)
				result := Value{Num: CatVal{Rat: *v, Unit: to}, Den: oneCatVal(), Display: 10}
				return result, nil
			}
			// Rat is already in base units — just change display unit
			val.Num.Unit = n.Unit.Num
			val.Den.Unit = n.Unit.Den
			return val, nil
		}
		// First unit attachment — convert to base units (except offset-based like temperature)
		eff := val.effectiveRat()
		if n.Unit.HasOffset() {
			// Offset-based units (temperature): store in display units, not base
			return Value{Num: CatVal{Rat: *new(big.Rat).Set(eff), Unit: n.Unit.Num}, Den: oneCatVal()}, nil
		}
		var numRat big.Rat
		numRat.Set(eff)
		if n.Unit.Num != numUnit {
			numRat.Mul(&numRat, &n.Unit.Num.ToBase)
		}
		var denRat big.Rat
		denRat.SetInt64(1)
		if n.Unit.Den != numUnit {
			denRat.Mul(&denRat, &n.Unit.Den.ToBase)
		}
		return Value{Num: CatVal{Rat: numRat, Unit: n.Unit.Num}, Den: CatVal{Rat: denRat, Unit: n.Unit.Den}}, nil

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
	return tsVal(r), nil
}

func evalAMPM(n *AMPMExpr, env Env) (Value, error) {
	val, err := Eval(n.Expr, env)
	if err != nil {
		return Value{}, err
	}
	if !val.IsTimestamp() {
		return Value{}, &EvalError{Msg: "AM/PM can only be applied to time values"}
	}
	// Extract the hour from the UTC time
	unix := val.Num.Rat.Num().Int64() / val.Num.Rat.Denom().Int64()
	t := time.Unix(unix, 0).UTC()
	h := t.Hour()

	if n.IsPM {
		if h < 12 {
			val.Num.Add(&val.Num.Rat, new(big.Rat).SetInt64(12*3600))
		}
	} else {
		// AM
		if h == 12 {
			val.Num.Sub(&val.Num.Rat, new(big.Rat).SetInt64(12*3600))
		}
	}
	return val, nil
}

func evalTZExpr(n *TZExpr, env Env) (Value, error) {
	val, err := Eval(n.Expr, env)
	if err != nil {
		return Value{}, err
	}
	if !val.IsTimestamp() {
		return Value{}, &EvalError{Msg: "timezone can only be applied to time values"}
	}
	loc := LookupTimezone(n.TZ)
	if loc == nil {
		return Value{}, &EvalError{Msg: "unknown timezone: " + n.TZ}
	}
	if n.IsInput {
		_, offset := time.Unix(val.Num.Rat.Num().Int64()/val.Num.Rat.Denom().Int64(), 0).In(loc).Zone()
		adjustment := new(big.Rat).SetInt64(int64(offset))
		val.Num.Sub(&val.Num.Rat, adjustment)
	}
	val.Display = *loc
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
	if !val.IsEmpty() {
		return Value{}, &EvalError{Msg: n.Name + "() requires a dimensionless value"}
	}
	f, _ := val.effectiveRat().Float64()
	result := fn(f)
	r := new(big.Rat).SetFloat64(result)
	if r == nil {
		return Value{}, &EvalError{Msg: n.Name + "(): result out of range"}
	}
	v := dimless(r)
	v.Display = 10
	return v, nil
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
	if !a.IsEmpty() {
		return Value{}, &EvalError{Msg: n.Name + "() requires dimensionless values"}
	}
	if !b.IsEmpty() {
		return Value{}, &EvalError{Msg: n.Name + "() requires dimensionless values"}
	}
	af, _ := a.effectiveRat().Float64()
	bf, _ := b.effectiveRat().Float64()
	result := fn(af, bf)
	r := new(big.Rat).SetFloat64(result)
	if r == nil {
		return Value{}, &EvalError{Msg: n.Name + "(): result out of range"}
	}
	v := dimless(r)
	v.Display = 10
	return v, nil
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
		if !v.IsEmpty() {
			return Value{}, &EvalError{Msg: n.Name + "() requires dimensionless values"}
		}
		vals[i], _ = v.effectiveRat().Float64()
	}
	result := fn(vals[0], vals[1], vals[2])
	r := new(big.Rat).SetFloat64(result)
	if r == nil {
		return Value{}, &EvalError{Msg: n.Name + "(): result out of range"}
	}
	v := dimless(r)
	v.Display = 10
	return v, nil
}

func evalTimeExtract(n *FuncCall, env Env, extract func(time.Time) int) (Value, error) {
	if len(n.Args) != 1 {
		return Value{}, &EvalError{Msg: n.Name + "() takes 1 argument"}
	}
	val, err := Eval(n.Args[0], env)
	if err != nil {
		return Value{}, err
	}
	if !val.IsTimestamp() {
		return Value{}, &EvalError{Msg: n.Name + "() requires a time value"}
	}
	unix := val.Num.Rat.Num().Int64() / val.Num.Rat.Denom().Int64()
	loc := time.UTC
	if tz, ok := val.Display.(time.Location); ok {
		loc = &tz
	}
	t := time.Unix(unix, 0).In(loc)
	r := new(big.Rat).SetInt64(int64(extract(t)))
	return dimless(r), nil
}

// ratFloor returns floor(x) as an integer-valued *big.Rat.
func ratFloor(x *big.Rat) *big.Rat {
	q := new(big.Int).Quo(x.Num(), x.Denom())
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
	if !val.IsEmpty() {
		return Value{}, &EvalError{Msg: n.Name + "() requires a dimensionless value"}
	}
	eff := val.effectiveRat()
	v := dimless(fn(eff))
	v.Display = 10
	return v, nil
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
	if !a.IsEmpty() {
		return Value{}, &EvalError{Msg: n.Name + "() requires dimensionless values"}
	}
	if !b.IsEmpty() {
		return Value{}, &EvalError{Msg: n.Name + "() requires dimensionless values"}
	}
	aEff := a.effectiveRat()
	bEff := b.effectiveRat()
	v := dimless(fn(aEff, bEff))
	v.Display = 10
	return v, nil
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
	if !base.IsEmpty() {
		return Value{}, &EvalError{Msg: "pow() requires dimensionless values"}
	}
	if !exp.IsEmpty() {
		return Value{}, &EvalError{Msg: "pow() requires dimensionless values"}
	}
	baseR := base.effectiveRat()
	expR := exp.effectiveRat()
	// If exponent is integer, use exact rational arithmetic
	if expR.IsInt() {
		e := expR.Num().Int64()
		neg := e < 0
		if neg {
			e = -e
		}
		num := new(big.Int).Set(baseR.Num())
		den := new(big.Int).Set(baseR.Denom())
		num.Exp(num, big.NewInt(e), nil)
		den.Exp(den, big.NewInt(e), nil)
		r := new(big.Rat).SetFrac(num, den)
		if neg {
			if r.Sign() == 0 {
				return Value{}, &EvalError{Msg: "pow(): division by zero"}
			}
			r.Inv(r)
		}
		v := dimless(r)
		v.Display = 10
		return v, nil
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
		return tsVal(r), nil

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
			eff := v.effectiveRat()
			if !eff.IsInt() {
				return Value{}, &EvalError{Msg: "date() arguments must be integers"}
			}
			vals[i] = int(eff.Num().Int64())
		}
		var t time.Time
		if len(vals) == 3 {
			t = time.Date(vals[0], time.Month(vals[1]), vals[2], 0, 0, 0, 0, time.UTC)
		} else {
			t = time.Date(vals[0], time.Month(vals[1]), vals[2], vals[3], vals[4], vals[5], 0, time.UTC)
		}
		r := new(big.Rat).SetInt64(t.Unix())
		return tsVal(r), nil

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
			eff := v.effectiveRat()
			if !eff.IsInt() {
				return Value{}, &EvalError{Msg: "time() arguments must be integers"}
			}
			vals[i] = int(eff.Num().Int64())
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
		return tsVal(r), nil

	case "__to_unix":
		if len(n.Args) != 1 {
			return Value{}, &EvalError{Msg: "to unix requires a value"}
		}
		val, err := Eval(n.Args[0], env)
		if err != nil {
			return Value{}, err
		}
		if !val.IsTimestamp() {
			return Value{}, &EvalError{Msg: "to unix requires a time value"}
		}
		v := dimless(val.effectiveRat())
		v.Display = 10
		return v, nil

	case "__to_hex", "__to_bin", "__to_oct":
		if len(n.Args) != 1 {
			return Value{}, &EvalError{Msg: "to " + n.Name[5:] + " requires a value"}
		}
		val, err := Eval(n.Args[0], env)
		if err != nil {
			return Value{}, err
		}
		if !val.DisplayRat().IsInt() {
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
		if val.IsTimestamp() {
			// Strip timestamp, keep as dimensionless
			v := dimless(val.effectiveRat())
			v.Display = base
			return v, nil
		}
		val.Display = base
		return val, nil

	case "unix":
		if len(n.Args) != 1 {
			return Value{}, &EvalError{Msg: "unix() takes 1 argument"}
		}
		val, err := Eval(n.Args[0], env)
		if err != nil {
			return Value{}, err
		}
		if !val.IsEmpty() {
			return Value{}, &EvalError{Msg: "unix() value must be dimensionless"}
		}
		eff := val.effectiveRat()
		r := autoDetectUnixPrecision(eff)
		return tsVal(r), nil

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
			return pmt * (math.Pow(1+rate, nf) - 1) / rate
		})
	case "pv":
		return evalFinanceFunc3(n, env, func(rate, nf, pmt float64) float64 {
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
	v := new(big.Rat).Set(r)
	if v.Sign() < 0 {
		v.Neg(v)
	}

	threshMs := new(big.Rat).SetInt64(1e12)
	threshUs := new(big.Rat).SetInt64(1e15)
	threshNs := new(big.Rat).SetInt64(1e18)

	result := new(big.Rat).Set(r)
	if v.Cmp(threshMs) < 0 {
		return result
	} else if v.Cmp(threshUs) < 0 {
		return result.Quo(result, new(big.Rat).SetInt64(1000))
	} else if v.Cmp(threshNs) < 0 {
		return result.Quo(result, new(big.Rat).SetInt64(1e6))
	}
	return result.Quo(result, new(big.Rat).SetInt64(1e9))
}

// EvalLine lexes, parses, and evaluates a single line.
func EvalLine(line string, env Env) (Value, error) {
	tokens := Lex(line)

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

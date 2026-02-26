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
type Env map[string]CompoundValue

// tsVal builds a timestamp CompoundValue from a rational (unix seconds).
func tsVal(r *big.Rat) CompoundValue {
	return simpleVal(Value{Rat: new(big.Rat).Set(r), Unit: tsUnit})
}

// Eval evaluates an AST node in the given environment.
func Eval(node Node, env Env) (CompoundValue, error) {
	if node == nil {
		return CompoundValue{}, &EvalError{Msg: "empty expression"}
	}

	switch n := node.(type) {
	case *NumberLit:
		return dimless(n.Value), nil

	case *VarRef:
		v, ok := env[n.Name]
		if !ok {
			// Try looking up as a unit — bare unit word implies 1
			if u := LookupUnit(n.Name); u != nil {
				return simpleVal(Value{Rat: new(big.Rat).Set(toBaseRat(*u)), Unit: *u}), nil
			}
			// Built-in constants
			switch n.Name {
			case "pi":
				v := dimless(new(big.Rat).Set(piRat))
				v.Num.Unit = decUnit
				return v, nil
			case "e":
				v := dimless(new(big.Rat).Set(eRat))
				v.Num.Unit = decUnit
				return v, nil
			case "c":
				return CompoundValue{
					Num: Value{Rat: new(big.Rat).Set(cRat), Unit: *LookupUnit("m")},
					Den: Value{Rat: new(big.Rat).SetInt64(1), Unit: *LookupUnit("s")},
				}, nil
			}
			return CompoundValue{}, &EvalError{Msg: "undefined variable: " + n.Name}
		}
		return v, nil

	case *BinaryExpr:
		left, err := Eval(n.Left, env)
		if err != nil {
			return CompoundValue{}, err
		}
		right, err := Eval(n.Right, env)
		if err != nil {
			return CompoundValue{}, err
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
		case TOKEN_STARSTAR:
			return valPow(left, right)
		case TOKEN_AMP:
			return valBitwise(left, right, "and")
		case TOKEN_PIPE:
			return valBitwise(left, right, "or")
		case TOKEN_CARET:
			return valBitwise(left, right, "xor")
		case TOKEN_LSHIFT:
			return valShift(left, right, "left")
		case TOKEN_RSHIFT:
			return valShift(left, right, "right")
		default:
			return CompoundValue{}, &EvalError{Msg: "unknown operator"}
		}

	case *UnaryExpr:
		operand, err := Eval(n.Operand, env)
		if err != nil {
			return CompoundValue{}, err
		}
		if n.Op == TOKEN_MINUS {
			return valNeg(operand), nil
		}
		if n.Op == TOKEN_TILDE {
			return valBitwiseNot(operand)
		}
		return CompoundValue{}, &EvalError{Msg: "unknown unary operator"}

	case *PercentExpr:
		val, err := Eval(n.Expr, env)
		if err != nil {
			return CompoundValue{}, err
		}
		r := new(big.Rat).Quo(val.effectiveRat(), new(big.Rat).SetInt64(100))
		return dimless(r), nil

	case *FactorialExpr:
		val, err := Eval(n.Expr, env)
		if err != nil {
			return CompoundValue{}, err
		}
		return valFactorial(val)

	case *UnitExpr:
		val, err := Eval(n.Expr, env)
		if err != nil {
			return CompoundValue{}, err
		}
		valCU := val.CompoundUnit()
		if !valCU.IsEmpty() {
			// Already has a unit — convert if compatible
			if !valCU.Compatible(n.Unit) {
				return CompoundValue{}, &EvalError{Msg: "cannot convert " + valCU.String() + " to " + n.Unit.String()}
			}
			// Block cross-currency conversion (no exchange rates)
			if valCU.Num.Category == UnitCurrency && n.Unit.Num.Category == UnitCurrency &&
				valCU.Num.Short != n.Unit.Num.Short {
				return CompoundValue{}, &EvalError{Msg: "__forex__"}
			}
			// Offset-based conversion (temperature)
			if valCU.HasOffset() || n.Unit.HasOffset() {
				if val.Den.Unit.Category != UnitNumber || n.Unit.Den.Category != UnitNumber {
					return CompoundValue{}, &EvalError{Msg: "temperature units cannot be used in compound units"}
				}
				from := val.Num.Unit
				to := n.Unit.Num
				eff := val.effectiveRat()
				v := new(big.Rat).Set(eff)
				v.Add(v, preOffsetRat(from))
				v.Mul(v, toBaseRat(from))
				v.Quo(v, toBaseRat(to))
				v.Sub(v, preOffsetRat(to))
				return simpleVal(Value{Rat: v, Unit: to}), nil
			}
			// Rat is already in base units — just change display unit
			val.Num.Unit = n.Unit.Num
			val.Den.Unit = n.Unit.Den
			return val, nil
		}
		// First unit attachment — convert to base units (except offset-based like temperature)
		eff := val.effectiveRat()
		if n.Unit.HasOffset() {
			return simpleVal(Value{Rat: new(big.Rat).Set(eff), Unit: n.Unit.Num}), nil
		}
		numRat := new(big.Rat).Set(eff)
		if n.Unit.Num.Category != UnitNumber {
			numRat.Mul(numRat, toBaseRat(n.Unit.Num))
		}
		denRat := new(big.Rat).SetInt64(1)
		if n.Unit.Den.Category != UnitNumber {
			denRat.Mul(denRat, toBaseRat(n.Unit.Den))
		}
		return CompoundValue{
			Num: Value{Rat: numRat, Unit: n.Unit.Num},
			Den: Value{Rat: denRat, Unit: n.Unit.Den},
		}, nil

	case *Assignment:
		val, err := Eval(n.Expr, env)
		if err != nil {
			return CompoundValue{}, err
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
		return CompoundValue{}, &EvalError{Msg: "unknown node type"}
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

func evalTimeLit(raw string) (CompoundValue, error) {
	var h, m, s int
	var err error
	if len(raw) > 5 {
		_, err = fmt.Sscanf(raw, "%d:%d:%d", &h, &m, &s)
	} else {
		_, err = fmt.Sscanf(raw, "%d:%d", &h, &m)
	}
	if err != nil {
		return CompoundValue{}, &EvalError{Msg: "invalid time: " + raw}
	}
	if h < 0 || h > 23 || m < 0 || m > 59 || s < 0 || s > 59 {
		return CompoundValue{}, &EvalError{Msg: "invalid time: " + raw}
	}
	now := time.Now().UTC()
	t := time.Date(now.Year(), now.Month(), now.Day(), h, m, s, 0, time.UTC)
	return tsVal(new(big.Rat).SetInt64(t.Unix())), nil
}

func evalAMPM(n *AMPMExpr, env Env) (CompoundValue, error) {
	val, err := Eval(n.Expr, env)
	if err != nil {
		return CompoundValue{}, err
	}
	if !val.IsTimestamp() {
		return CompoundValue{}, &EvalError{Msg: "AM/PM can only be applied to time values"}
	}
	unix := val.Num.Rat.Num().Int64() / val.Num.Rat.Denom().Int64()
	t := time.Unix(unix, 0).UTC()
	h := t.Hour()

	if n.IsPM {
		if h < 12 {
			val.Num.Rat = new(big.Rat).Add(val.Num.Rat, new(big.Rat).SetInt64(12*3600))
		}
	} else {
		if h == 12 {
			val.Num.Rat = new(big.Rat).Sub(val.Num.Rat, new(big.Rat).SetInt64(12*3600))
		}
	}
	return val, nil
}

func evalTZExpr(n *TZExpr, env Env) (CompoundValue, error) {
	val, err := Eval(n.Expr, env)
	if err != nil {
		return CompoundValue{}, err
	}
	if !val.IsTimestamp() {
		return CompoundValue{}, &EvalError{Msg: "timezone can only be applied to time values"}
	}
	tzUnit, ok := LookupTZUnit(n.TZ)
	if !ok {
		return CompoundValue{}, &EvalError{Msg: "unknown timezone: " + n.TZ}
	}
	if n.IsInput {
		loc := tzUnit.PreOffset.(time.Location)
		_, offset := time.Unix(val.Num.Rat.Num().Int64()/val.Num.Rat.Denom().Int64(), 0).In(&loc).Zone()
		val.Num.Rat = new(big.Rat).Sub(val.Num.Rat, new(big.Rat).SetInt64(int64(offset)))
	}
	val.Num.Unit = tzUnit
	return val, nil
}

func evalMathFunc1(n *FuncCall, env Env, fn func(float64) float64) (CompoundValue, error) {
	if len(n.Args) != 1 {
		return CompoundValue{}, &EvalError{Msg: n.Name + "() takes 1 argument"}
	}
	val, err := Eval(n.Args[0], env)
	if err != nil {
		return CompoundValue{}, err
	}
	if !val.IsEmpty() {
		return CompoundValue{}, &EvalError{Msg: n.Name + "() requires a dimensionless value"}
	}
	f, _ := val.effectiveRat().Float64()
	result := fn(f)
	r := new(big.Rat).SetFloat64(result)
	if r == nil {
		return CompoundValue{}, &EvalError{Msg: n.Name + "(): result out of range"}
	}
	v := dimless(r)
	v.Num.Unit = decUnit
	return v, nil
}

func evalMathFunc2(n *FuncCall, env Env, fn func(float64, float64) float64) (CompoundValue, error) {
	if len(n.Args) != 2 {
		return CompoundValue{}, &EvalError{Msg: n.Name + "() takes 2 arguments"}
	}
	a, err := Eval(n.Args[0], env)
	if err != nil {
		return CompoundValue{}, err
	}
	b, err := Eval(n.Args[1], env)
	if err != nil {
		return CompoundValue{}, err
	}
	if !a.IsEmpty() {
		return CompoundValue{}, &EvalError{Msg: n.Name + "() requires dimensionless values"}
	}
	if !b.IsEmpty() {
		return CompoundValue{}, &EvalError{Msg: n.Name + "() requires dimensionless values"}
	}
	af, _ := a.effectiveRat().Float64()
	bf, _ := b.effectiveRat().Float64()
	result := fn(af, bf)
	r := new(big.Rat).SetFloat64(result)
	if r == nil {
		return CompoundValue{}, &EvalError{Msg: n.Name + "(): result out of range"}
	}
	v := dimless(r)
	v.Num.Unit = decUnit
	return v, nil
}

func evalFinanceFunc3(n *FuncCall, env Env, fn func(float64, float64, float64) float64) (CompoundValue, error) {
	if len(n.Args) != 3 {
		return CompoundValue{}, &EvalError{Msg: n.Name + "() takes 3 arguments"}
	}
	vals := make([]float64, 3)
	for i, arg := range n.Args {
		v, err := Eval(arg, env)
		if err != nil {
			return CompoundValue{}, err
		}
		if !v.IsEmpty() {
			return CompoundValue{}, &EvalError{Msg: n.Name + "() requires dimensionless values"}
		}
		vals[i], _ = v.effectiveRat().Float64()
	}
	result := fn(vals[0], vals[1], vals[2])
	r := new(big.Rat).SetFloat64(result)
	if r == nil {
		return CompoundValue{}, &EvalError{Msg: n.Name + "(): result out of range"}
	}
	v := dimless(r)
	v.Num.Unit = decUnit
	return v, nil
}

func evalTimeExtract(n *FuncCall, env Env, extract func(time.Time) int) (CompoundValue, error) {
	if len(n.Args) != 1 {
		return CompoundValue{}, &EvalError{Msg: n.Name + "() takes 1 argument"}
	}
	val, err := Eval(n.Args[0], env)
	if err != nil {
		return CompoundValue{}, err
	}
	if !val.IsTimestamp() {
		return CompoundValue{}, &EvalError{Msg: n.Name + "() requires a time value"}
	}
	unix := val.Num.Rat.Num().Int64() / val.Num.Rat.Denom().Int64()
	loc := time.UTC
	if tz, ok := val.Num.Unit.PreOffset.(time.Location); ok {
		loc = &tz
	}
	t := time.Unix(unix, 0).In(loc)
	return dimless(new(big.Rat).SetInt64(int64(extract(t)))), nil
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
	return new(big.Rat).Neg(ratFloor(new(big.Rat).Neg(x)))
}

// ratRound returns round(x) using banker's rounding (round half to even).
func ratRound(x *big.Rat) *big.Rat {
	f := ratFloor(new(big.Rat).Set(x))
	frac := new(big.Rat).Sub(new(big.Rat).Set(x), f)
	half := new(big.Rat).SetFrac64(1, 2)
	cmp := frac.Cmp(half)
	if x.Sign() >= 0 {
		if cmp < 0 {
			return f
		}
		if cmp > 0 {
			return new(big.Rat).Add(f, new(big.Rat).SetInt64(1))
		}
		// Exactly 0.5: round to nearest even
		floorInt := new(big.Int).Div(f.Num(), f.Denom())
		if new(big.Int).And(floorInt, big.NewInt(1)).Sign() == 0 {
			return f // floor is even, keep it
		}
		return new(big.Rat).Add(f, new(big.Rat).SetInt64(1))
	}
	// Negative: work with absolute value
	absX := new(big.Rat).Neg(x)
	pos := ratRound(absX)
	return new(big.Rat).Neg(pos)
}

func evalRatFunc1(n *FuncCall, env Env, fn func(*big.Rat) *big.Rat) (CompoundValue, error) {
	if len(n.Args) != 1 {
		return CompoundValue{}, &EvalError{Msg: n.Name + "() takes 1 argument"}
	}
	val, err := Eval(n.Args[0], env)
	if err != nil {
		return CompoundValue{}, err
	}
	if !val.IsEmpty() {
		return CompoundValue{}, &EvalError{Msg: n.Name + "() requires a dimensionless value"}
	}
	return dimless(fn(val.effectiveRat())), nil
}

func evalRatFunc2(n *FuncCall, env Env, fn func(*big.Rat, *big.Rat) *big.Rat) (CompoundValue, error) {
	if len(n.Args) != 2 {
		return CompoundValue{}, &EvalError{Msg: n.Name + "() takes 2 arguments"}
	}
	a, err := Eval(n.Args[0], env)
	if err != nil {
		return CompoundValue{}, err
	}
	b, err := Eval(n.Args[1], env)
	if err != nil {
		return CompoundValue{}, err
	}
	if !a.IsEmpty() {
		return CompoundValue{}, &EvalError{Msg: n.Name + "() requires dimensionless values"}
	}
	if !b.IsEmpty() {
		return CompoundValue{}, &EvalError{Msg: n.Name + "() requires dimensionless values"}
	}
	return dimless(fn(a.effectiveRat(), b.effectiveRat())), nil
}

func evalPow(n *FuncCall, env Env) (CompoundValue, error) {
	if len(n.Args) != 2 {
		return CompoundValue{}, &EvalError{Msg: "pow() takes 2 arguments"}
	}
	base, err := Eval(n.Args[0], env)
	if err != nil {
		return CompoundValue{}, err
	}
	exp, err := Eval(n.Args[1], env)
	if err != nil {
		return CompoundValue{}, err
	}
	if !base.IsEmpty() {
		return CompoundValue{}, &EvalError{Msg: "pow() requires dimensionless values"}
	}
	if !exp.IsEmpty() {
		return CompoundValue{}, &EvalError{Msg: "pow() requires dimensionless values"}
	}
	baseR := base.effectiveRat()
	expR := exp.effectiveRat()
	if expR.IsInt() {
		e := expR.Num().Int64()
		neg := e < 0
		if neg {
			e = -e
		}
		num := new(big.Int).Exp(new(big.Int).Set(baseR.Num()), big.NewInt(e), nil)
		den := new(big.Int).Exp(new(big.Int).Set(baseR.Denom()), big.NewInt(e), nil)
		r := new(big.Rat).SetFrac(num, den)
		if neg {
			if r.Sign() == 0 {
				return CompoundValue{}, &EvalError{Msg: "pow(): division by zero"}
			}
			r.Inv(r)
		}
		return dimless(r), nil
	}
	return evalMathFunc2(n, env, math.Pow)
}

// valPow computes left ** right using exact rational arithmetic for integer exponents.
func valPow(left, right CompoundValue) (CompoundValue, error) {
	if !left.IsEmpty() {
		return CompoundValue{}, &EvalError{Msg: "** requires dimensionless values"}
	}
	if !right.IsEmpty() {
		return CompoundValue{}, &EvalError{Msg: "** requires dimensionless values"}
	}
	baseR := left.effectiveRat()
	expR := right.effectiveRat()
	if expR.IsInt() {
		e := expR.Num().Int64()
		neg := e < 0
		if neg {
			e = -e
		}
		num := new(big.Int).Exp(new(big.Int).Set(baseR.Num()), big.NewInt(e), nil)
		den := new(big.Int).Exp(new(big.Int).Set(baseR.Denom()), big.NewInt(e), nil)
		r := new(big.Rat).SetFrac(num, den)
		if neg {
			if r.Sign() == 0 {
				return CompoundValue{}, &EvalError{Msg: "**: division by zero"}
			}
			r.Inv(r)
		}
		return dimless(r), nil
	}
	// Non-integer exponent: use float
	bf, _ := baseR.Float64()
	ef, _ := expR.Float64()
	result := math.Pow(bf, ef)
	r := new(big.Rat).SetFloat64(result)
	if r == nil {
		return CompoundValue{}, &EvalError{Msg: "**: result out of range"}
	}
	v := dimless(r)
	v.Num.Unit = decUnit
	return v, nil
}

// valBitwise performs bitwise AND, OR, XOR on two integer values.
func valBitwise(left, right CompoundValue, op string) (CompoundValue, error) {
	lr := left.DisplayRat()
	rr := right.DisplayRat()
	if !lr.IsInt() || !rr.IsInt() {
		return CompoundValue{}, &EvalError{Msg: op + " requires integer operands"}
	}
	a := new(big.Int).Set(lr.Num())
	b := new(big.Int).Set(rr.Num())
	var result *big.Int
	switch op {
	case "and":
		result = new(big.Int).And(a, b)
	case "or":
		result = new(big.Int).Or(a, b)
	case "xor":
		result = new(big.Int).Xor(a, b)
	}
	return dimless(new(big.Rat).SetInt(result)), nil
}

// valShift performs left/right bit shift.
func valShift(left, right CompoundValue, dir string) (CompoundValue, error) {
	lr := left.DisplayRat()
	rr := right.DisplayRat()
	if !lr.IsInt() || !rr.IsInt() {
		return CompoundValue{}, &EvalError{Msg: "shift requires integer operands"}
	}
	a := new(big.Int).Set(lr.Num())
	n := rr.Num().Int64()
	if n < 0 {
		return CompoundValue{}, &EvalError{Msg: "shift count must be non-negative"}
	}
	var result *big.Int
	switch dir {
	case "left":
		result = new(big.Int).Lsh(a, uint(n))
	case "right":
		result = new(big.Int).Rsh(a, uint(n))
	}
	return dimless(new(big.Rat).SetInt(result)), nil
}

// valBitwiseNot performs bitwise NOT (~) on an integer value.
func valBitwiseNot(val CompoundValue) (CompoundValue, error) {
	r := val.DisplayRat()
	if !r.IsInt() {
		return CompoundValue{}, &EvalError{Msg: "~ requires an integer operand"}
	}
	result := new(big.Int).Not(r.Num())
	return dimless(new(big.Rat).SetInt(result)), nil
}

// valFactorial computes n! for a non-negative integer.
func valFactorial(val CompoundValue) (CompoundValue, error) {
	r := val.DisplayRat()
	if !r.IsInt() {
		return CompoundValue{}, &EvalError{Msg: "! requires a non-negative integer"}
	}
	n := r.Num().Int64()
	if r.Sign() < 0 {
		return CompoundValue{}, &EvalError{Msg: "! requires a non-negative integer"}
	}
	if n > 10000 {
		return CompoundValue{}, &EvalError{Msg: "! argument too large"}
	}
	result := new(big.Int).SetInt64(1)
	for i := int64(2); i <= n; i++ {
		result.Mul(result, big.NewInt(i))
	}
	return dimless(new(big.Rat).SetInt(result)), nil
}

func evalFuncCall(n *FuncCall, env Env) (CompoundValue, error) {
	switch n.Name {
	case "now":
		if len(n.Args) != 0 {
			return CompoundValue{}, &EvalError{Msg: "now() takes no arguments"}
		}
		return tsVal(new(big.Rat).SetInt64(time.Now().Unix())), nil

	case "date":
		if len(n.Args) != 3 && len(n.Args) != 6 {
			return CompoundValue{}, &EvalError{Msg: "date() takes 3 or 6 arguments"}
		}
		vals := make([]int, len(n.Args))
		for i, arg := range n.Args {
			v, err := Eval(arg, env)
			if err != nil {
				return CompoundValue{}, err
			}
			eff := v.effectiveRat()
			if !eff.IsInt() {
				return CompoundValue{}, &EvalError{Msg: "date() arguments must be integers"}
			}
			vals[i] = int(eff.Num().Int64())
		}
		var t time.Time
		if len(vals) == 3 {
			t = time.Date(vals[0], time.Month(vals[1]), vals[2], 0, 0, 0, 0, time.UTC)
		} else {
			t = time.Date(vals[0], time.Month(vals[1]), vals[2], vals[3], vals[4], vals[5], 0, time.UTC)
		}
		return tsVal(new(big.Rat).SetInt64(t.Unix())), nil

	case "time":
		if len(n.Args) != 2 && len(n.Args) != 3 {
			return CompoundValue{}, &EvalError{Msg: "time() takes 2 or 3 arguments"}
		}
		vals := make([]int, len(n.Args))
		for i, arg := range n.Args {
			v, err := Eval(arg, env)
			if err != nil {
				return CompoundValue{}, err
			}
			eff := v.effectiveRat()
			if !eff.IsInt() {
				return CompoundValue{}, &EvalError{Msg: "time() arguments must be integers"}
			}
			vals[i] = int(eff.Num().Int64())
		}
		h, m := vals[0], vals[1]
		s := 0
		if len(vals) == 3 {
			s = vals[2]
		}
		if h < 0 || h > 23 || m < 0 || m > 59 || s < 0 || s > 59 {
			return CompoundValue{}, &EvalError{Msg: "invalid time"}
		}
		now := time.Now().UTC()
		tt := time.Date(now.Year(), now.Month(), now.Day(), h, m, s, 0, time.UTC)
		return tsVal(new(big.Rat).SetInt64(tt.Unix())), nil

	case "__to_unix":
		if len(n.Args) != 1 {
			return CompoundValue{}, &EvalError{Msg: "to unix requires a value"}
		}
		val, err := Eval(n.Args[0], env)
		if err != nil {
			return CompoundValue{}, err
		}
		if !val.IsTimestamp() {
			return CompoundValue{}, &EvalError{Msg: "to unix requires a time value"}
		}
		v := dimless(val.effectiveRat())
		v.Num.Unit = decUnit
		return v, nil

	case "__to_hex", "__to_bin", "__to_oct":
		if len(n.Args) != 1 {
			return CompoundValue{}, &EvalError{Msg: "to " + n.Name[5:] + " requires a value"}
		}
		val, err := Eval(n.Args[0], env)
		if err != nil {
			return CompoundValue{}, err
		}
		if !val.DisplayRat().IsInt() {
			return CompoundValue{}, &EvalError{Msg: "to " + n.Name[5:] + " requires an integer"}
		}
		var baseUnit Unit
		switch n.Name {
		case "__to_hex":
			baseUnit = hexUnit
		case "__to_bin":
			baseUnit = binUnit
		case "__to_oct":
			baseUnit = octUnit
		}
		v := dimless(val.DisplayRat())
		v.Num.Unit = baseUnit
		return v, nil

	case "unix":
		if len(n.Args) != 1 {
			return CompoundValue{}, &EvalError{Msg: "unix() takes 1 argument"}
		}
		val, err := Eval(n.Args[0], env)
		if err != nil {
			return CompoundValue{}, err
		}
		if !val.IsEmpty() {
			return CompoundValue{}, &EvalError{Msg: "unix() value must be dimensionless"}
		}
		return tsVal(autoDetectUnixPrecision(val.effectiveRat())), nil

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

	case "num":
		if len(n.Args) != 1 {
			return CompoundValue{}, &EvalError{Msg: "num() takes 1 argument"}
		}
		val, err := Eval(n.Args[0], env)
		if err != nil {
			return CompoundValue{}, err
		}
		return dimless(val.DisplayRat()), nil

	case "__to_hms":
		if len(n.Args) != 1 {
			return CompoundValue{}, &EvalError{Msg: "to hms requires a value"}
		}
		val, err := Eval(n.Args[0], env)
		if err != nil {
			return CompoundValue{}, err
		}
		if !isSimpleTimeUnit(val) && !val.IsEmpty() {
			return CompoundValue{}, &EvalError{Msg: "to hms requires a time or dimensionless value"}
		}
		// Convert to seconds (effectiveRat is already in base = seconds for time units)
		secs := val.effectiveRat()
		v := dimless(new(big.Rat).Set(secs))
		v.Num.Unit = hmsUnit
		return v, nil

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
		return CompoundValue{}, &EvalError{Msg: "unknown function: " + n.Name}
	}
}

// autoDetectUnixPrecision converts a unix timestamp to seconds, auto-detecting
// if the input is in seconds, milliseconds, microseconds, or nanoseconds.
func autoDetectUnixPrecision(r *big.Rat) *big.Rat {
	v := new(big.Rat).Abs(r)

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
func EvalLine(line string, env Env) (CompoundValue, error) {
	tokens := Lex(line)

	allEOF := true
	for _, t := range tokens {
		if t.Type != TOKEN_EOF {
			allEOF = false
			break
		}
	}
	if allEOF {
		return CompoundValue{}, &EvalError{Msg: ""}
	}

	node, err := Parse(tokens)
	if err != nil {
		return CompoundValue{}, err
	}
	if node == nil {
		return CompoundValue{}, &EvalError{Msg: ""}
	}
	return Eval(node, env)
}

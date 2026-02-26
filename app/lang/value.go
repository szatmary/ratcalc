package lang

import (
	"fmt"
	"math/big"
	"strings"
	"time"
)

// Value represents a rational number with an optional compound unit.
//
// Rendering is controlled by the Base field:
//
//	0  — default: integers and simple fractions (denom ≤ 1000) as fractions,
//	     otherwise truncated decimal
//	10 — force decimal: always render as decimal, never as a fraction
//	     (used by "to unix" so timestamps display as numbers)
//	2  — binary:  prefix 0b, integer only
//	8  — octal:   prefix 0o, integer only
//	16 — hex:     prefix 0x, integer only
type Value struct {
	Rat    *big.Rat
	Unit   *CompoundUnit  // nil means dimensionless
	IsTime bool           // when true, Rat holds unix seconds and Unit must be nil
	TZ     *time.Location // display timezone for time values; nil means UTC
	Base   int            // display base (see Value doc)
}

// String formats the value for display.
// Simple fractions (denom <= 1000, not integer) display as fractions.
// Otherwise display as decimal. Appends unit string if present.
func (v Value) String() string {
	if v.IsTime {
		sec := v.Rat.Num().Int64() / v.Rat.Denom().Int64()
		t := time.Unix(sec, 0).UTC()
		if v.TZ != nil {
			t = t.In(v.TZ)
			_, offset := t.Zone()
			sign := "+"
			if offset < 0 {
				sign = "-"
				offset = -offset
			}
			h := offset / 3600
			m := (offset % 3600) / 60
			return fmt.Sprintf("%s %s%02d%02d", t.Format("2006-01-02 15:04:05"), sign, h, m)
		}
		return t.Format("2006-01-02 15:04:05 +0000")
	}
	if v.Base != 0 && v.Base != 10 && v.Rat.IsInt() {
		s := formatIntBase(v.Rat.Num(), v.Base)
		if v.Unit != nil {
			us := v.Unit.String()
			if us != "" {
				s += " " + us
			}
		}
		return s
	}
	var s string
	if v.Base == 10 || hasTimeUnit(v.Unit) {
		s = formatDecimal(v.Rat)
	} else {
		s = formatRat(v.Rat)
	}
	if v.Unit != nil {
		us := v.Unit.String()
		if us != "" {
			s += " " + us
		}
	}
	return s
}

func formatIntBase(n *big.Int, base int) string {
	neg := n.Sign() < 0
	abs := new(big.Int).Set(n)
	if neg {
		abs.Neg(abs)
	}
	var prefix string
	switch base {
	case 16:
		prefix = "0x"
	case 2:
		prefix = "0b"
	case 8:
		prefix = "0o"
	}
	s := prefix + abs.Text(base)
	if neg {
		s = "-" + s
	}
	return s
}

// formatDecimal always renders as a decimal number, never as a fraction.
func formatDecimal(r *big.Rat) string {
	if r.IsInt() {
		return r.Num().String()
	}
	return ratToDecimal(r, 10)
}

func formatRat(r *big.Rat) string {
	if r.IsInt() {
		return r.Num().String()
	}

	denom := new(big.Int).Set(r.Denom())
	thousand := big.NewInt(1000)

	if denom.Cmp(thousand) <= 0 {
		return r.RatString()
	}

	// Convert to decimal string
	return ratToDecimal(r, 10)
}

// ratToDecimal converts a rational to a decimal string with up to `prec` digits
// after the decimal point.
func ratToDecimal(r *big.Rat, prec int) string {
	// Sign
	neg := r.Sign() < 0
	num := new(big.Int).Set(r.Num())
	den := new(big.Int).Set(r.Denom())
	if neg {
		num.Neg(num)
	}

	// Integer part
	intPart := new(big.Int)
	remainder := new(big.Int)
	intPart.DivMod(num, den, remainder)

	if remainder.Sign() == 0 {
		s := intPart.String()
		if neg {
			s = "-" + s
		}
		return s
	}

	// Fractional digits
	ten := big.NewInt(10)
	var digits []byte
	for i := 0; i < prec; i++ {
		remainder.Mul(remainder, ten)
		digit := new(big.Int)
		digit.DivMod(remainder, den, remainder)
		digits = append(digits, byte('0'+digit.Int64()))
		if remainder.Sign() == 0 {
			break
		}
	}

	// Trim trailing zeros
	s := strings.TrimRight(string(digits), "0")
	result := intPart.String() + "." + s
	if neg {
		result = "-" + result
	}
	return result
}

// EvalError represents an evaluation error.
type EvalError struct {
	Msg string
}

func (e *EvalError) Error() string {
	return e.Msg
}

// Arithmetic operations on Values

func valAdd(a, b Value) (Value, error) {
	// Time guards
	if a.IsTime && b.IsTime {
		return Value{}, &EvalError{Msg: "cannot add two times"}
	}
	if a.IsTime && !b.IsTime {
		if b.Unit != nil && isSimpleTimeUnit(b.Unit) {
			// time + duration = time
			secs := durationToSeconds(b)
			r := new(big.Rat).Add(a.Rat, secs)
			return Value{Rat: r, IsTime: true, TZ: a.TZ}, nil
		}
		return Value{}, &EvalError{Msg: "cannot add to time: use a time unit (s, min, hr, d, etc.)"}
	}
	if !a.IsTime && b.IsTime {
		if a.Unit != nil && isSimpleTimeUnit(a.Unit) {
			// duration + time = time
			secs := durationToSeconds(a)
			r := new(big.Rat).Add(secs, b.Rat)
			return Value{Rat: r, IsTime: true, TZ: b.TZ}, nil
		}
		return Value{}, &EvalError{Msg: "cannot add to time: use a time unit (s, min, hr, d, etc.)"}
	}

	au, bu := a.Unit, b.Unit
	if au == nil && bu == nil {
		r := new(big.Rat).Add(a.Rat, b.Rat)
		return Value{Rat: r}, nil
	}
	if au == nil || bu == nil {
		return Value{}, &EvalError{Msg: "cannot add values with and without units"}
	}
	if !au.Compatible(bu) {
		return Value{}, &EvalError{Msg: fmt.Sprintf("cannot add %s and %s", au.String(), bu.String())}
	}
	// Convert b's value using conversion factors for each unit pair
	factor := compoundConversionFactor(bu, au)
	bConverted := new(big.Rat).Mul(b.Rat, factor)
	r := new(big.Rat).Add(a.Rat, bConverted)
	return Value{Rat: r, Unit: au}, nil
}

func valSub(a, b Value) (Value, error) {
	// Time guards
	if a.IsTime && b.IsTime {
		// time - time = duration in seconds
		r := new(big.Rat).Sub(a.Rat, b.Rat)
		return Value{Rat: r, Unit: SimpleUnit(SecondsUnit())}, nil
	}
	if a.IsTime && !b.IsTime {
		if b.Unit != nil && isSimpleTimeUnit(b.Unit) {
			// time - duration = time
			secs := durationToSeconds(b)
			r := new(big.Rat).Sub(a.Rat, secs)
			return Value{Rat: r, IsTime: true, TZ: a.TZ}, nil
		}
		return Value{}, &EvalError{Msg: "cannot subtract from time: use a time unit (s, min, hr, d, etc.)"}
	}
	if b.IsTime {
		return Value{}, &EvalError{Msg: "cannot subtract time from non-time value"}
	}

	au, bu := a.Unit, b.Unit
	if au == nil && bu == nil {
		r := new(big.Rat).Sub(a.Rat, b.Rat)
		return Value{Rat: r}, nil
	}
	if au == nil || bu == nil {
		return Value{}, &EvalError{Msg: "cannot subtract values with and without units"}
	}
	if !au.Compatible(bu) {
		return Value{}, &EvalError{Msg: fmt.Sprintf("cannot subtract %s and %s", au.String(), bu.String())}
	}
	factor := compoundConversionFactor(bu, au)
	bConverted := new(big.Rat).Mul(b.Rat, factor)
	r := new(big.Rat).Sub(a.Rat, bConverted)
	return Value{Rat: r, Unit: au}, nil
}

func valMul(a, b Value) (Value, error) {
	if a.IsTime || b.IsTime {
		return Value{}, &EvalError{Msg: "cannot multiply time values"}
	}
	r := new(big.Rat).Mul(a.Rat, b.Rat)
	u := mergeUnits(a.Unit, b.Unit)
	return Value{Rat: r, Unit: u}, nil
}

func valDiv(a, b Value) (Value, error) {
	if a.IsTime || b.IsTime {
		return Value{}, &EvalError{Msg: "cannot divide time values"}
	}
	if b.Rat.Sign() == 0 {
		return Value{}, &EvalError{Msg: "division by zero"}
	}
	r := new(big.Rat).Quo(a.Rat, b.Rat)
	// For division: a.Num+b.Den → new Num, a.Den+b.Num → new Den
	u := divideUnits(a.Unit, b.Unit)
	return Value{Rat: r, Unit: u}, nil
}

func valNeg(a Value) Value {
	r := new(big.Rat).Neg(a.Rat)
	return Value{Rat: r, Unit: a.Unit, IsTime: a.IsTime, TZ: a.TZ}
}

// mergeUnits merges compound units for multiplication.
// Concatenates Num lists and Den lists.
func mergeUnits(a, b *CompoundUnit) *CompoundUnit {
	if a == nil && b == nil {
		return nil
	}
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	result := &CompoundUnit{
		Num: append(append([]*Unit{}, a.Num...), b.Num...),
		Den: append(append([]*Unit{}, a.Den...), b.Den...),
	}
	if result.IsEmpty() {
		return nil
	}
	return result
}

// divideUnits creates the compound unit for division.
// a.Num+b.Den → new Num, a.Den+b.Num → new Den.
func divideUnits(a, b *CompoundUnit) *CompoundUnit {
	if a == nil && b == nil {
		return nil
	}
	var aNum, aDen, bNum, bDen []*Unit
	if a != nil {
		aNum = a.Num
		aDen = a.Den
	}
	if b != nil {
		bNum = b.Num
		bDen = b.Den
	}
	result := &CompoundUnit{
		Num: append(append([]*Unit{}, aNum...), bDen...),
		Den: append(append([]*Unit{}, aDen...), bNum...),
	}
	if result.IsEmpty() {
		return nil
	}
	return result
}

// hasTimeUnit returns true if any unit in the compound unit is a time-category unit.
func hasTimeUnit(u *CompoundUnit) bool {
	if u == nil {
		return false
	}
	for _, unit := range u.Num {
		if unit.Category == UnitTime {
			return true
		}
	}
	for _, unit := range u.Den {
		if unit.Category == UnitTime {
			return true
		}
	}
	return false
}

// isSimpleTimeUnit returns true if the compound unit is a single numerator unit
// in the UnitTime category with no denominator. This identifies durations.
func isSimpleTimeUnit(u *CompoundUnit) bool {
	return u != nil && len(u.Num) == 1 && len(u.Den) == 0 && u.Num[0].Category == UnitTime
}

// durationToSeconds converts a duration value to seconds using its unit's ToBase factor.
func durationToSeconds(v Value) *big.Rat {
	return new(big.Rat).Mul(v.Rat, v.Unit.Num[0].ToBase)
}

// compoundConversionFactor computes the overall conversion factor to convert
// a value from compound unit `from` to compound unit `to`.
// For each positional pair in Num: multiply by (from.ToBase / to.ToBase)
// For each positional pair in Den: multiply by (to.ToBase / from.ToBase)
// (denominator units convert inversely)
func compoundConversionFactor(from, to *CompoundUnit) *big.Rat {
	factor := new(big.Rat).SetInt64(1)
	for i := range from.Num {
		// factor *= from.Num[i].ToBase / to.Num[i].ToBase
		f := new(big.Rat).Quo(from.Num[i].ToBase, to.Num[i].ToBase)
		factor.Mul(factor, f)
	}
	for i := range from.Den {
		// factor *= to.Den[i].ToBase / from.Den[i].ToBase
		f := new(big.Rat).Quo(to.Den[i].ToBase, from.Den[i].ToBase)
		factor.Mul(factor, f)
	}
	return factor
}

package lang

import (
	"fmt"
	"math/big"
	"strings"
	"time"
)

// CatVal pairs a rational value (in base units) with its unit.
// The Rat holds the magnitude in base-unit terms. Unit is nil for dimensionless.
type CatVal struct {
	big.Rat
	Unit *Unit // nil = dimensionless
}

// Value represents a rational number with optional compound units.
// The effective numeric value is Num.Rat / Den.Rat.
//
// Display controls rendering:
//
//	nil             — default: integers and simple fractions (denom ≤ 1000) as
//	                  fractions, otherwise truncated decimal
//	time.Location   — timestamp timezone (value, not pointer)
//	int             — forced display base:
//	                    10 = decimal (never fraction)
//	                    2  = binary  (0b prefix, integer only)
//	                    8  = octal   (0o prefix, integer only)
//	                    16 = hex     (0x prefix, integer only)
type Value struct {
	Num     CatVal
	Den     CatVal
	Display any // display hint (see Value doc)
}

// oneCatVal returns a CatVal with Rat=1 and Unit=nil (dimensionless 1).
func oneCatVal() CatVal {
	var cv CatVal
	cv.Rat.SetInt64(1)
	return cv
}

// dimless creates a dimensionless Value from a rational.
func dimless(r *big.Rat) Value {
	var v Value
	v.Num.Rat.Set(r)
	v.Den = oneCatVal()
	return v
}

// IsTimestamp returns true if the value represents an absolute point in time.
func (v Value) IsTimestamp() bool {
	return v.Num.Unit == tsUnit && v.Den.Unit == nil
}

// CompoundUnit reconstructs the CompoundUnit for display.
func (v Value) CompoundUnit() CompoundUnit {
	return CompoundUnit{Num: v.Num.Unit, Den: v.Den.Unit}
}

// IsEmpty returns true if both units are nil (dimensionless).
func (v Value) IsEmpty() bool {
	return v.Num.Unit == nil && v.Den.Unit == nil
}

// effectiveRat returns Num.Rat / Den.Rat as a new *big.Rat.
// If Den.Rat is zero (zero-value Value), returns a copy of Num.Rat.
func (v Value) effectiveRat() *big.Rat {
	if v.Den.Rat.Sign() == 0 {
		return new(big.Rat).Set(&v.Num.Rat)
	}
	return new(big.Rat).Quo(&v.Num.Rat, &v.Den.Rat)
}

// Sign returns the sign of the effective value.
func (v Value) Sign() int {
	return v.effectiveRat().Sign()
}

// DisplayRat returns the value converted from base units to display units.
func (v Value) DisplayRat() *big.Rat {
	if v.IsTimestamp() {
		return v.effectiveRat()
	}
	r := v.effectiveRat()
	// Convert numerator from base to display units
	if v.Num.Unit != nil && !v.Num.Unit.HasOffset() {
		r.Quo(r, &v.Num.Unit.ToBase)
	}
	// Convert denominator from base to display units (inverse)
	if v.Den.Unit != nil && !v.Den.Unit.HasOffset() {
		r.Mul(r, &v.Den.Unit.ToBase)
	}
	return r
}

// String formats the value for display.
func (v Value) String() string {
	if v.IsTimestamp() {
		sec := v.Num.Rat.Num().Int64() / v.Num.Rat.Denom().Int64()
		t := time.Unix(sec, 0).UTC()
		if loc, ok := v.Display.(time.Location); ok {
			t = t.In(&loc)
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
	dr := v.DisplayRat()
	base, _ := v.Display.(int)
	cu := v.CompoundUnit()
	if base != 0 && base != 10 && dr.IsInt() {
		s := formatIntBase(dr.Num(), base)
		if us := cu.String(); us != "" {
			s += " " + us
		}
		return s
	}
	var s string
	if base == 10 || hasTimeUnit(cu) {
		s = formatDecimal(dr)
	} else {
		s = formatRat(dr)
	}
	if us := cu.String(); us != "" {
		s += " " + us
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

// unitEqual returns true if two Values have the same compound unit structure.
func unitEqual(a, b Value) bool {
	return a.Num.Unit == b.Num.Unit && a.Den.Unit == b.Den.Unit
}

// Arithmetic operations on Values

func valAdd(a, b Value) (Value, error) {
	// Time guards
	if a.IsTimestamp() && b.IsTimestamp() {
		return Value{}, &EvalError{Msg: "cannot add two times"}
	}
	if a.IsTimestamp() && !b.IsTimestamp() {
		if isSimpleTimeUnit(b) {
			// time + duration = time
			secs := durationToSeconds(b)
			var r big.Rat
			r.Add(&a.Num.Rat, secs)
			return Value{Num: CatVal{Rat: r, Unit: tsUnit}, Den: oneCatVal(), Display: a.Display}, nil
		}
		return Value{}, &EvalError{Msg: "cannot add to time: use a time unit (s, min, hr, d, etc.)"}
	}
	if !a.IsTimestamp() && b.IsTimestamp() {
		if isSimpleTimeUnit(a) {
			// duration + time = time
			secs := durationToSeconds(a)
			var r big.Rat
			r.Add(secs, &b.Num.Rat)
			return Value{Num: CatVal{Rat: r, Unit: tsUnit}, Den: oneCatVal(), Display: b.Display}, nil
		}
		return Value{}, &EvalError{Msg: "cannot add to time: use a time unit (s, min, hr, d, etc.)"}
	}

	au, bu := a.CompoundUnit(), b.CompoundUnit()
	if au.IsEmpty() && bu.IsEmpty() {
		// Both dimensionless: cross-multiply to add a.Num/a.Den + b.Num/b.Den
		r := new(big.Rat).Add(a.effectiveRat(), b.effectiveRat())
		return dimless(r), nil
	}
	if au.IsEmpty() || bu.IsEmpty() {
		return Value{}, &EvalError{Msg: "cannot add values with and without units"}
	}
	if !au.Compatible(bu) {
		return Value{}, &EvalError{Msg: fmt.Sprintf("cannot add %s and %s", au.String(), bu.String())}
	}
	// Temperature (offset-based): values stored in display units, need conversion
	if au.HasOffset() || bu.HasOffset() {
		factor := compoundConversionFactor(bu, au)
		bEff := b.effectiveRat()
		bConverted := new(big.Rat).Mul(bEff, factor)
		aEff := a.effectiveRat()
		r := new(big.Rat).Add(aEff, bConverted)
		return Value{Num: CatVal{Rat: *r, Unit: a.Num.Unit}, Den: CatVal{Rat: *new(big.Rat).SetInt64(1), Unit: a.Den.Unit}}, nil
	}
	// Both in base units — add effective rats, keep a's units
	r := new(big.Rat).Add(a.effectiveRat(), b.effectiveRat())
	return Value{Num: CatVal{Rat: *r, Unit: a.Num.Unit}, Den: CatVal{Rat: *new(big.Rat).SetInt64(1), Unit: a.Den.Unit}}, nil
}

func valSub(a, b Value) (Value, error) {
	// Time guards
	if a.IsTimestamp() && b.IsTimestamp() {
		// time - time = duration in seconds
		var r big.Rat
		r.Sub(&a.Num.Rat, &b.Num.Rat)
		return Value{Num: CatVal{Rat: r, Unit: SecondsUnit()}, Den: oneCatVal()}, nil
	}
	if a.IsTimestamp() && !b.IsTimestamp() {
		if isSimpleTimeUnit(b) {
			// time - duration = time
			secs := durationToSeconds(b)
			var r big.Rat
			r.Sub(&a.Num.Rat, secs)
			return Value{Num: CatVal{Rat: r, Unit: tsUnit}, Den: oneCatVal(), Display: a.Display}, nil
		}
		return Value{}, &EvalError{Msg: "cannot subtract from time: use a time unit (s, min, hr, d, etc.)"}
	}
	if b.IsTimestamp() {
		return Value{}, &EvalError{Msg: "cannot subtract time from non-time value"}
	}

	au, bu := a.CompoundUnit(), b.CompoundUnit()
	if au.IsEmpty() && bu.IsEmpty() {
		r := new(big.Rat).Sub(a.effectiveRat(), b.effectiveRat())
		return dimless(r), nil
	}
	if au.IsEmpty() || bu.IsEmpty() {
		return Value{}, &EvalError{Msg: "cannot subtract values with and without units"}
	}
	if !au.Compatible(bu) {
		return Value{}, &EvalError{Msg: fmt.Sprintf("cannot subtract %s and %s", au.String(), bu.String())}
	}
	// Temperature (offset-based)
	if au.HasOffset() || bu.HasOffset() {
		factor := compoundConversionFactor(bu, au)
		bEff := b.effectiveRat()
		bConverted := new(big.Rat).Mul(bEff, factor)
		aEff := a.effectiveRat()
		r := new(big.Rat).Sub(aEff, bConverted)
		return Value{Num: CatVal{Rat: *r, Unit: a.Num.Unit}, Den: CatVal{Rat: *new(big.Rat).SetInt64(1), Unit: a.Den.Unit}}, nil
	}
	r := new(big.Rat).Sub(a.effectiveRat(), b.effectiveRat())
	return Value{Num: CatVal{Rat: *r, Unit: a.Num.Unit}, Den: CatVal{Rat: *new(big.Rat).SetInt64(1), Unit: a.Den.Unit}}, nil
}

func valMul(a, b Value) (Value, error) {
	if a.IsTimestamp() || b.IsTimestamp() {
		return Value{}, &EvalError{Msg: "cannot multiply time values"}
	}
	// Multiply: result.Num = a.Num * b.Num, result.Den = a.Den * b.Den
	var numRat, denRat big.Rat
	numRat.Mul(&a.Num.Rat, &b.Num.Rat)
	denRat.Mul(&a.Den.Rat, &b.Den.Rat)

	numUnit, denUnit, err := cancelUnits(a.Num.Unit, b.Num.Unit, a.Den.Unit, b.Den.Unit)
	if err != nil {
		return Value{}, err
	}
	return Value{Num: CatVal{Rat: numRat, Unit: numUnit}, Den: CatVal{Rat: denRat, Unit: denUnit}}, nil
}

func valDiv(a, b Value) (Value, error) {
	if a.IsTimestamp() || b.IsTimestamp() {
		return Value{}, &EvalError{Msg: "cannot divide time values"}
	}
	if b.effectiveRat().Sign() == 0 {
		return Value{}, &EvalError{Msg: "division by zero"}
	}
	// Division: a/b = (a.Num*b.Den) / (a.Den*b.Num)
	var numRat, denRat big.Rat
	numRat.Mul(&a.Num.Rat, &b.Den.Rat)
	denRat.Mul(&a.Den.Rat, &b.Num.Rat)

	numUnit, denUnit, err := cancelUnits(a.Num.Unit, b.Den.Unit, a.Den.Unit, b.Num.Unit)
	if err != nil {
		return Value{}, err
	}
	return Value{Num: CatVal{Rat: numRat, Unit: numUnit}, Den: CatVal{Rat: denRat, Unit: denUnit}}, nil
}

// cancelUnits implements category cancellation for mul/div.
// numA, numB are the two units going into the numerator.
// denA, denB are the two units going into the denominator.
// If a category appears on both sides, it cancels.
// After cancellation, each side must have at most 1 category.
func cancelUnits(numA, numB, denA, denB *Unit) (numUnit, denUnit *Unit, err error) {
	// Collect non-nil units per side
	type catUnit struct {
		cat  UnitCategory
		unit *Unit
	}
	var nums, dens []catUnit
	if numA != nil {
		nums = append(nums, catUnit{numA.Category, numA})
	}
	if numB != nil {
		nums = append(nums, catUnit{numB.Category, numB})
	}
	if denA != nil {
		dens = append(dens, catUnit{denA.Category, denA})
	}
	if denB != nil {
		dens = append(dens, catUnit{denB.Category, denB})
	}

	// Cancel matching categories across num/den
	for i := 0; i < len(nums); i++ {
		for j := 0; j < len(dens); j++ {
			if nums[i].cat == dens[j].cat {
				// Cancel: remove from both
				nums = append(nums[:i], nums[i+1:]...)
				dens = append(dens[:j], dens[j+1:]...)
				i--
				break
			}
		}
	}

	if len(nums) > 1 {
		return nil, nil, &EvalError{Msg: "cannot combine units"}
	}
	if len(dens) > 1 {
		return nil, nil, &EvalError{Msg: "cannot combine units"}
	}

	if len(nums) == 1 {
		numUnit = nums[0].unit
	}
	if len(dens) == 1 {
		denUnit = dens[0].unit
	}
	return numUnit, denUnit, nil
}

func valNeg(a Value) Value {
	var r big.Rat
	r.Neg(&a.Num.Rat)
	return Value{Num: CatVal{Rat: r, Unit: a.Num.Unit}, Den: a.Den, Display: a.Display}
}

// hasTimeUnit returns true if any unit in the value is a time-category unit.
func hasTimeUnit(u CompoundUnit) bool {
	if u.Num != nil && u.Num.Category == UnitTime {
		return true
	}
	if u.Den != nil && u.Den.Category == UnitTime {
		return true
	}
	return false
}

// isSimpleTimeUnit returns true if the value has a single numerator unit
// in the UnitTime category with no denominator unit. This identifies durations.
func isSimpleTimeUnit(v Value) bool {
	return v.Num.Unit != nil && v.Num.Unit.Category == UnitTime && v.Den.Unit == nil
}

// durationToSeconds returns the duration in seconds.
// The Num.Rat is already in base units (seconds), so just return effective rat.
func durationToSeconds(v Value) *big.Rat {
	return v.effectiveRat()
}

// compoundConversionFactor computes the conversion factor from compound unit `from` to `to`.
func compoundConversionFactor(from, to CompoundUnit) *big.Rat {
	factor := new(big.Rat).SetInt64(1)
	if from.Num != nil && to.Num != nil {
		f := new(big.Rat).Quo(&from.Num.ToBase, &to.Num.ToBase)
		factor.Mul(factor, f)
	}
	if from.Den != nil && to.Den != nil {
		f := new(big.Rat).Quo(&to.Den.ToBase, &from.Den.ToBase)
		factor.Mul(factor, f)
	}
	return factor
}

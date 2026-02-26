package lang

import (
	"fmt"
	"math/big"
	"strings"
	"time"
)

// Value pairs a rational value (in base units) with its unit.
// The Rat holds the magnitude in base-unit terms. Unit is numUnit for dimensionless.
type Value struct {
	Rat  *big.Rat
	Unit Unit // numUnit = dimensionless
}

// CompoundValue represents a rational number with optional compound units.
// The effective numeric value is Num.Rat / Den.Rat.
type CompoundValue struct {
	Num Value
	Den Value
}

// oneVal returns a Value with Rat=1 and Unit=numUnit (dimensionless 1).
func oneVal() Value {
	return Value{Rat: new(big.Rat).SetInt64(1), Unit: numUnit}
}

// dimless creates a dimensionless CompoundValue from a rational.
func dimless(r *big.Rat) CompoundValue {
	return CompoundValue{
		Num: Value{Rat: new(big.Rat).Set(r), Unit: numUnit},
		Den: oneVal(),
	}
}

// simpleVal creates a CompoundValue from a single Value (Den = 1 dimensionless).
func simpleVal(v Value) CompoundValue {
	return CompoundValue{Num: v, Den: oneVal()}
}

// IsTimestamp returns true if the value represents an absolute point in time.
func (v CompoundValue) IsTimestamp() bool {
	return v.Num.Unit.Category == UnitTimestamp && v.Den.Unit.Category == UnitNumber
}

// CompoundUnit reconstructs the CompoundUnit for display.
func (v CompoundValue) CompoundUnit() CompoundUnit {
	return CompoundUnit{Num: v.Num.Unit, Den: v.Den.Unit}
}

// IsEmpty returns true if both units are dimensionless.
func (v CompoundValue) IsEmpty() bool {
	return v.Num.Unit.Category == UnitNumber && v.Den.Unit.Category == UnitNumber
}

// effectiveRat returns Num.Rat / Den.Rat as a new *big.Rat.
// If Den.Rat is nil or zero (zero-value CompoundValue), returns a copy of Num.Rat.
func (v CompoundValue) effectiveRat() *big.Rat {
	if v.Num.Rat == nil {
		return new(big.Rat)
	}
	if v.Den.Rat == nil || v.Den.Rat.Sign() == 0 {
		return new(big.Rat).Set(v.Num.Rat)
	}
	return new(big.Rat).Quo(v.Num.Rat, v.Den.Rat)
}

// Sign returns the sign of the effective value.
func (v CompoundValue) Sign() int {
	return v.effectiveRat().Sign()
}

// displayBase returns the display base if the numerator unit encodes one (int ToBase).
func displayBase(v CompoundValue) (int, bool) {
	b, ok := v.Num.Unit.ToBase.(int)
	return b, ok
}

// DisplayRat returns the value converted from base units to display units.
func (v CompoundValue) DisplayRat() *big.Rat {
	if v.Num.Unit.Category == UnitTimestamp {
		return v.effectiveRat()
	}
	r := v.effectiveRat()
	// Convert numerator from base to display units
	if v.Num.Unit.Category != UnitNumber && !v.Num.Unit.HasOffset() {
		r.Quo(r, toBaseRat(v.Num.Unit))
	}
	// Convert denominator from base to display units (inverse)
	if v.Den.Unit.Category != UnitNumber && !v.Den.Unit.HasOffset() {
		r.Mul(r, toBaseRat(v.Den.Unit))
	}
	return r
}

// String formats the value for display.
func (v CompoundValue) String() string {
	if v.Num.Unit.Category == UnitTimestamp {
		sec := v.Num.Rat.Num().Int64() / v.Num.Rat.Denom().Int64()
		t := time.Unix(sec, 0).UTC()
		if loc, ok := v.Num.Unit.PreOffset.(time.Location); ok {
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
	// Check for HMS display
	if v.Num.Unit.ToBase == "hms" {
		return formatHMS(v.effectiveRat())
	}

	// Check for currency display
	if v.Num.Unit.Category == UnitCurrency && v.Den.Unit.Category == UnitNumber {
		return formatCurrency(v)
	}

	dr := v.DisplayRat()
	cu := v.CompoundUnit()

	// Check for base display (hex/bin/oct)
	if base, ok := displayBase(v); ok && base != 10 && dr.IsInt() {
		return formatIntBase(dr.Num(), base)
	}

	var s string
	_, isBase := displayBase(v)
	if isBase || hasTimeUnit(cu) || cu.HasOffset() {
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

// MaxDisplayLen is the max character width for a result in the gutter.
// Set by the UI layer based on actual measured width.
var MaxDisplayLen = 32

func formatRat(r *big.Rat) string {
	if r.IsInt() {
		s := r.Num().String()
		if len(s) <= MaxDisplayLen {
			return s
		}
		return formatSci(r)
	}

	// Try fraction form first
	frac := r.RatString()
	if len(frac) <= MaxDisplayLen {
		return frac
	}

	// Try decimal — but reject if it lost all significance (e.g. "0.")
	dec := ratToDecimal(r, 10)
	if len(dec) <= MaxDisplayLen && !strings.HasSuffix(dec, ".") {
		return dec
	}

	return formatSci(r)
}

// formatHMS formats a rational number of seconds as "Xh Ym Zs".
func formatHMS(r *big.Rat) string {
	neg := r.Sign() < 0
	abs := new(big.Rat).Abs(r)
	total := new(big.Int).Div(abs.Num(), abs.Denom())

	hours := new(big.Int).Div(total, big.NewInt(3600))
	rem := new(big.Int).Mod(total, big.NewInt(3600))
	mins := new(big.Int).Div(rem, big.NewInt(60))
	secs := new(big.Int).Mod(rem, big.NewInt(60))

	var s string
	if hours.Sign() > 0 {
		s = hours.String() + "h "
	}
	if hours.Sign() > 0 || mins.Sign() > 0 {
		s += mins.String() + "m "
	}
	s += secs.String() + "s"
	if neg {
		s = "-" + s
	}
	return s
}

// formatCurrency formats a currency value with 2 decimal places.
// Uses symbol prefix for known currencies ($80.00, €50.00) and suffix for others (80.00 CAD).
func formatCurrency(v CompoundValue) string {
	dr := v.DisplayRat()

	// Round to 2 decimal places: multiply by 100, round, divide by 100
	scaled := new(big.Rat).Mul(dr, new(big.Rat).SetInt64(100))
	rounded := ratRound(scaled)
	cents := new(big.Int).Div(rounded.Num(), rounded.Denom())

	neg := cents.Sign() < 0
	absCents := new(big.Int).Abs(cents)

	intPart := new(big.Int).Div(absCents, big.NewInt(100))
	fracPart := new(big.Int).Mod(absCents, big.NewInt(100))

	numStr := fmt.Sprintf("%s.%02d", intPart.String(), fracPart.Int64())
	if neg {
		numStr = "-" + numStr
	}

	if sym, ok := currencySymbols[v.Num.Unit.Short]; ok {
		if neg {
			return "-" + sym + numStr[1:] // -$80.00
		}
		return sym + numStr
	}
	return numStr + " " + v.Num.Unit.Short
}

// formatSci formats a rational in scientific notation (e.g. 1.23e15).
func formatSci(r *big.Rat) string {
	f, _ := r.Float64()
	if f == 0 {
		return "0"
	}
	s := fmt.Sprintf("%e", f)
	// Trim trailing zeros in mantissa: 1.230000e+02 → 1.23e+02
	parts := strings.SplitN(s, "e", 2)
	if len(parts) == 2 {
		m := strings.TrimRight(parts[0], "0")
		m = strings.TrimRight(m, ".")
		s = m + "e" + parts[1]
	}
	return s
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

// unitEqual returns true if two CompoundValues have the same compound unit structure.
func unitEqual(a, b CompoundValue) bool {
	return a.Num.Unit.Category == b.Num.Unit.Category &&
		a.Num.Unit.Short == b.Num.Unit.Short &&
		a.Den.Unit.Category == b.Den.Unit.Category &&
		a.Den.Unit.Short == b.Den.Unit.Short
}

// Arithmetic operations on CompoundValues

func valAdd(a, b CompoundValue) (CompoundValue, error) {
	// Time guards
	if a.IsTimestamp() && b.IsTimestamp() {
		return CompoundValue{}, &EvalError{Msg: "cannot add two times"}
	}
	if a.IsTimestamp() && !b.IsTimestamp() {
		if isSimpleTimeUnit(b) {
			// time + duration = time
			secs := durationToSeconds(b)
			r := new(big.Rat).Add(a.Num.Rat, secs)
			return simpleVal(Value{Rat: r, Unit: a.Num.Unit}), nil
		}
		return CompoundValue{}, &EvalError{Msg: "cannot add to time: use a time unit (s, min, hr, d, etc.)"}
	}
	if !a.IsTimestamp() && b.IsTimestamp() {
		if isSimpleTimeUnit(a) {
			// duration + time = time
			secs := durationToSeconds(a)
			r := new(big.Rat).Add(secs, b.Num.Rat)
			return simpleVal(Value{Rat: r, Unit: b.Num.Unit}), nil
		}
		return CompoundValue{}, &EvalError{Msg: "cannot add to time: use a time unit (s, min, hr, d, etc.)"}
	}

	au, bu := a.CompoundUnit(), b.CompoundUnit()
	if au.IsEmpty() && bu.IsEmpty() {
		r := new(big.Rat).Add(a.effectiveRat(), b.effectiveRat())
		return dimless(r), nil
	}
	if au.IsEmpty() || bu.IsEmpty() {
		return CompoundValue{}, &EvalError{Msg: "cannot add values with and without units"}
	}
	if !au.Compatible(bu) {
		return CompoundValue{}, &EvalError{Msg: fmt.Sprintf("cannot add %s and %s", au.String(), bu.String())}
	}
	// Temperature (offset-based): values stored in display units, need conversion
	if au.HasOffset() || bu.HasOffset() {
		factor := compoundConversionFactor(bu, au)
		bConverted := new(big.Rat).Mul(b.effectiveRat(), factor)
		r := new(big.Rat).Add(a.effectiveRat(), bConverted)
		return CompoundValue{
			Num: Value{Rat: r, Unit: a.Num.Unit},
			Den: Value{Rat: new(big.Rat).SetInt64(1), Unit: a.Den.Unit},
		}, nil
	}
	// Both in base units — add effective rats, keep a's units
	r := new(big.Rat).Add(a.effectiveRat(), b.effectiveRat())
	return CompoundValue{
		Num: Value{Rat: r, Unit: a.Num.Unit},
		Den: Value{Rat: new(big.Rat).SetInt64(1), Unit: a.Den.Unit},
	}, nil
}

func valSub(a, b CompoundValue) (CompoundValue, error) {
	// Time guards
	if a.IsTimestamp() && b.IsTimestamp() {
		// time - time = duration in seconds
		r := new(big.Rat).Sub(a.Num.Rat, b.Num.Rat)
		return simpleVal(Value{Rat: r, Unit: *SecondsUnit()}), nil
	}
	if a.IsTimestamp() && !b.IsTimestamp() {
		if isSimpleTimeUnit(b) {
			// time - duration = time
			secs := durationToSeconds(b)
			r := new(big.Rat).Sub(a.Num.Rat, secs)
			return simpleVal(Value{Rat: r, Unit: a.Num.Unit}), nil
		}
		return CompoundValue{}, &EvalError{Msg: "cannot subtract from time: use a time unit (s, min, hr, d, etc.)"}
	}
	if b.IsTimestamp() {
		return CompoundValue{}, &EvalError{Msg: "cannot subtract time from non-time value"}
	}

	au, bu := a.CompoundUnit(), b.CompoundUnit()
	if au.IsEmpty() && bu.IsEmpty() {
		r := new(big.Rat).Sub(a.effectiveRat(), b.effectiveRat())
		return dimless(r), nil
	}
	if au.IsEmpty() || bu.IsEmpty() {
		return CompoundValue{}, &EvalError{Msg: "cannot subtract values with and without units"}
	}
	if !au.Compatible(bu) {
		return CompoundValue{}, &EvalError{Msg: fmt.Sprintf("cannot subtract %s and %s", au.String(), bu.String())}
	}
	// Temperature (offset-based)
	if au.HasOffset() || bu.HasOffset() {
		factor := compoundConversionFactor(bu, au)
		bConverted := new(big.Rat).Mul(b.effectiveRat(), factor)
		r := new(big.Rat).Sub(a.effectiveRat(), bConverted)
		return CompoundValue{
			Num: Value{Rat: r, Unit: a.Num.Unit},
			Den: Value{Rat: new(big.Rat).SetInt64(1), Unit: a.Den.Unit},
		}, nil
	}
	r := new(big.Rat).Sub(a.effectiveRat(), b.effectiveRat())
	return CompoundValue{
		Num: Value{Rat: r, Unit: a.Num.Unit},
		Den: Value{Rat: new(big.Rat).SetInt64(1), Unit: a.Den.Unit},
	}, nil
}

func valMul(a, b CompoundValue) (CompoundValue, error) {
	if a.IsTimestamp() || b.IsTimestamp() {
		return CompoundValue{}, &EvalError{Msg: "cannot multiply time values"}
	}
	numRat := new(big.Rat).Mul(a.Num.Rat, b.Num.Rat)
	denRat := new(big.Rat).Mul(a.Den.Rat, b.Den.Rat)

	numUnit, denUnit, err := cancelUnits(a.Num.Unit, b.Num.Unit, a.Den.Unit, b.Den.Unit)
	if err != nil {
		return CompoundValue{}, err
	}
	return CompoundValue{
		Num: Value{Rat: numRat, Unit: numUnit},
		Den: Value{Rat: denRat, Unit: denUnit},
	}, nil
}

func valDiv(a, b CompoundValue) (CompoundValue, error) {
	if a.IsTimestamp() || b.IsTimestamp() {
		return CompoundValue{}, &EvalError{Msg: "cannot divide time values"}
	}
	if b.effectiveRat().Sign() == 0 {
		return CompoundValue{}, &EvalError{Msg: "division by zero"}
	}
	numRat := new(big.Rat).Mul(a.Num.Rat, b.Den.Rat)
	denRat := new(big.Rat).Mul(a.Den.Rat, b.Num.Rat)

	numUnit, denUnit, err := cancelUnits(a.Num.Unit, b.Den.Unit, a.Den.Unit, b.Num.Unit)
	if err != nil {
		return CompoundValue{}, err
	}
	return CompoundValue{
		Num: Value{Rat: numRat, Unit: numUnit},
		Den: Value{Rat: denRat, Unit: denUnit},
	}, nil
}

// cancelUnits implements category cancellation for mul/div.
func cancelUnits(numA, numB, denA, denB Unit) (resNum, resDen Unit, err error) {
	type catUnit struct {
		cat  UnitCategory
		unit Unit
	}
	var nums, dens []catUnit
	if numA.Category != UnitNumber {
		nums = append(nums, catUnit{numA.Category, numA})
	}
	if numB.Category != UnitNumber {
		nums = append(nums, catUnit{numB.Category, numB})
	}
	if denA.Category != UnitNumber {
		dens = append(dens, catUnit{denA.Category, denA})
	}
	if denB.Category != UnitNumber {
		dens = append(dens, catUnit{denB.Category, denB})
	}

	// Cancel matching categories across num/den
	for i := 0; i < len(nums); i++ {
		for j := 0; j < len(dens); j++ {
			if nums[i].cat == dens[j].cat {
				nums = append(nums[:i], nums[i+1:]...)
				dens = append(dens[:j], dens[j+1:]...)
				i--
				break
			}
		}
	}

	if len(nums) > 1 {
		return numUnit, numUnit, &EvalError{Msg: "cannot combine units"}
	}
	if len(dens) > 1 {
		return numUnit, numUnit, &EvalError{Msg: "cannot combine units"}
	}

	resNum = numUnit
	resDen = numUnit
	if len(nums) == 1 {
		resNum = nums[0].unit
	}
	if len(dens) == 1 {
		resDen = dens[0].unit
	}
	return resNum, resDen, nil
}

func valNeg(a CompoundValue) CompoundValue {
	return CompoundValue{
		Num: Value{Rat: new(big.Rat).Neg(a.Num.Rat), Unit: a.Num.Unit},
		Den: a.Den,
	}
}

// hasTimeUnit returns true if any unit in the value is a time-category unit.
func hasTimeUnit(u CompoundUnit) bool {
	return u.Num.Category == UnitTime || u.Den.Category == UnitTime
}

// isSimpleTimeUnit returns true if the value has a single numerator unit
// in the UnitTime category with no denominator unit.
func isSimpleTimeUnit(v CompoundValue) bool {
	return v.Num.Unit.Category == UnitTime && v.Den.Unit.Category == UnitNumber
}

// durationToSeconds returns the duration in seconds.
func durationToSeconds(v CompoundValue) *big.Rat {
	return v.effectiveRat()
}

// compoundConversionFactor computes the conversion factor from compound unit `from` to `to`.
func compoundConversionFactor(from, to CompoundUnit) *big.Rat {
	factor := new(big.Rat).SetInt64(1)
	if from.Num.Category != UnitNumber && to.Num.Category != UnitNumber {
		f := new(big.Rat).Quo(toBaseRat(from.Num), toBaseRat(to.Num))
		factor.Mul(factor, f)
	}
	if from.Den.Category != UnitNumber && to.Den.Category != UnitNumber {
		f := new(big.Rat).Quo(toBaseRat(to.Den), toBaseRat(from.Den))
		factor.Mul(factor, f)
	}
	return factor
}

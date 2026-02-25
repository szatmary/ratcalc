package lang

import (
	"math/big"
	"strings"
)

// UnitCategory groups related units.
type UnitCategory int

const (
	UnitLength UnitCategory = iota
	UnitWeight
	UnitTime
	UnitVolume
)

// Unit defines a unit with its category and conversion factor to the base unit.
type Unit struct {
	Short    string
	Full     string       // full singular name (e.g. "meter")
	FullPl   string       // full plural name (e.g. "meters")
	Category UnitCategory
	// ToBase is the conversion factor: value_in_base = value * ToBase
	ToBase *big.Rat
}

func ratFromFloat(f float64) *big.Rat {
	return new(big.Rat).SetFloat64(f)
}

func ratFromFrac(num, denom int64) *big.Rat {
	return new(big.Rat).SetFrac64(num, denom)
}

var allUnits = []*Unit{
	// Length (base: meters)
	{Short: "mm", Full: "millimeter", FullPl: "millimeters", Category: UnitLength, ToBase: ratFromFrac(1, 1000)},
	{Short: "cm", Full: "centimeter", FullPl: "centimeters", Category: UnitLength, ToBase: ratFromFrac(1, 100)},
	{Short: "m", Full: "meter", FullPl: "meters", Category: UnitLength, ToBase: ratFromFrac(1, 1)},
	{Short: "km", Full: "kilometer", FullPl: "kilometers", Category: UnitLength, ToBase: ratFromFrac(1000, 1)},
	{Short: "in", Full: "inch", FullPl: "inches", Category: UnitLength, ToBase: ratFromFloat(0.0254)},
	{Short: "ft", Full: "foot", FullPl: "feet", Category: UnitLength, ToBase: ratFromFloat(0.3048)},
	{Short: "yd", Full: "yard", FullPl: "yards", Category: UnitLength, ToBase: ratFromFloat(0.9144)},
	{Short: "mi", Full: "mile", FullPl: "miles", Category: UnitLength, ToBase: ratFromFloat(1609.344)},

	// Weight (base: grams)
	{Short: "mg", Full: "milligram", FullPl: "milligrams", Category: UnitWeight, ToBase: ratFromFrac(1, 1000)},
	{Short: "g", Full: "gram", FullPl: "grams", Category: UnitWeight, ToBase: ratFromFrac(1, 1)},
	{Short: "kg", Full: "kilogram", FullPl: "kilograms", Category: UnitWeight, ToBase: ratFromFrac(1000, 1)},
	{Short: "oz", Full: "ounce", FullPl: "ounces", Category: UnitWeight, ToBase: ratFromFloat(28.3495)},
	{Short: "lb", Full: "pound", FullPl: "pounds", Category: UnitWeight, ToBase: ratFromFloat(453.592)},

	// Time (base: seconds)
	{Short: "ms", Full: "millisecond", FullPl: "milliseconds", Category: UnitTime, ToBase: ratFromFrac(1, 1000)},
	{Short: "s", Full: "second", FullPl: "seconds", Category: UnitTime, ToBase: ratFromFrac(1, 1)},
	{Short: "min", Full: "minute", FullPl: "minutes", Category: UnitTime, ToBase: ratFromFrac(60, 1)},
	{Short: "hr", Full: "hour", FullPl: "hours", Category: UnitTime, ToBase: ratFromFrac(3600, 1)},
	{Short: "d", Full: "day", FullPl: "days", Category: UnitTime, ToBase: ratFromFrac(86400, 1)},
	{Short: "wk", Full: "week", FullPl: "weeks", Category: UnitTime, ToBase: ratFromFrac(604800, 1)},
	{Short: "yr", Full: "year", FullPl: "years", Category: UnitTime, ToBase: ratFromFrac(31557600, 1)},

	// Volume (base: milliliters)
	{Short: "mL", Full: "milliliter", FullPl: "milliliters", Category: UnitVolume, ToBase: ratFromFrac(1, 1)},
	{Short: "L", Full: "liter", FullPl: "liters", Category: UnitVolume, ToBase: ratFromFrac(1000, 1)},
	{Short: "floz", Full: "floz", FullPl: "floz", Category: UnitVolume, ToBase: ratFromFloat(29.5735)},
	{Short: "cup", Full: "cup", FullPl: "cups", Category: UnitVolume, ToBase: ratFromFloat(236.588)},
	{Short: "pt", Full: "pint", FullPl: "pints", Category: UnitVolume, ToBase: ratFromFloat(473.176)},
	{Short: "qt", Full: "quart", FullPl: "quarts", Category: UnitVolume, ToBase: ratFromFloat(946.353)},
	{Short: "gal", Full: "gallon", FullPl: "gallons", Category: UnitVolume, ToBase: ratFromFloat(3785.41)},
}

// unitLookup maps short names, full singular, and full plural to unit pointers.
var unitLookup map[string]*Unit

func init() {
	unitLookup = make(map[string]*Unit, len(allUnits)*3)
	for _, u := range allUnits {
		unitLookup[u.Short] = u
		unitLookup[u.Full] = u
		unitLookup[u.FullPl] = u
	}
}

// LookupUnit looks up a unit by short name, full name, or plural name.
// Returns nil if not found.
func LookupUnit(name string) *Unit {
	return unitLookup[name]
}

// SecondsUnit returns the "s" unit entry.
func SecondsUnit() *Unit {
	return unitLookup["s"]
}

// Convert converts a rational value from one unit to another within the same category.
// Returns the converted value in terms of the target unit.
func Convert(val *big.Rat, from, to *Unit) (*big.Rat, error) {
	if from.Category != to.Category {
		return nil, &EvalError{Msg: "cannot convert between " + from.Short + " and " + to.Short}
	}
	// val_base = val * from.ToBase
	// result = val_base / to.ToBase
	result := new(big.Rat).Mul(val, from.ToBase)
	result.Quo(result, to.ToBase)
	return result, nil
}

// CompoundUnit represents a compound unit like mi/gal or m*s.
// A nil *CompoundUnit means dimensionless.
type CompoundUnit struct {
	Num []*Unit // numerator units
	Den []*Unit // denominator units
}

// SimpleUnit creates a CompoundUnit from a single unit.
func SimpleUnit(u *Unit) *CompoundUnit {
	return &CompoundUnit{Num: []*Unit{u}}
}

// IsEmpty returns true if there are no units in either num or den.
func (c *CompoundUnit) IsEmpty() bool {
	return len(c.Num) == 0 && len(c.Den) == 0
}

// String formats the compound unit for display.
// Examples: "m", "m*s", "mi/gal", "m*kg/s*s"
func (c *CompoundUnit) String() string {
	if c == nil || c.IsEmpty() {
		return ""
	}
	var parts []string
	for _, u := range c.Num {
		parts = append(parts, u.Short)
	}
	num := strings.Join(parts, "*")
	if len(c.Den) == 0 {
		return num
	}
	parts = parts[:0]
	for _, u := range c.Den {
		parts = append(parts, u.Short)
	}
	den := strings.Join(parts, "*")
	if num == "" {
		num = "1"
	}
	return num + "/" + den
}

// Compatible checks whether two compound units are compatible for add/sub.
// They must have the same number of Num and Den units, with matching categories
// at each position.
func (c *CompoundUnit) Compatible(other *CompoundUnit) bool {
	if c == nil && other == nil {
		return true
	}
	if c == nil || other == nil {
		return false
	}
	if len(c.Num) != len(other.Num) || len(c.Den) != len(other.Den) {
		return false
	}
	for i := range c.Num {
		if c.Num[i].Category != other.Num[i].Category {
			return false
		}
	}
	for i := range c.Den {
		if c.Den[i].Category != other.Den[i].Category {
			return false
		}
	}
	return true
}

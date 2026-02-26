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
	UnitTemperature
	UnitPressure
	UnitForce
	UnitEnergy
	UnitPower
	UnitVoltage
	UnitCurrent
	UnitResistance
	UnitData
)

// Unit defines a unit with its category and conversion factor to the base unit.
type Unit struct {
	Short    string
	Full     string       // full singular name (e.g. "meter")
	FullPl   string       // full plural name (e.g. "meters")
	Category UnitCategory
	// ToBase is the conversion factor: value_in_base = (value + PreOffset) * ToBase
	ToBase *big.Rat
	// PreOffset is added before multiplying by ToBase. Used for temperature.
	// nil means 0.
	PreOffset *big.Rat
}

// HasOffset returns true if this unit uses an offset-based conversion.
func (u *Unit) HasOffset() bool {
	return u.PreOffset != nil && u.PreOffset.Sign() != 0
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
	{Short: "in", Full: "inch", FullPl: "inches", Category: UnitLength, ToBase: ratFromFrac(127, 5000)},
	{Short: "ft", Full: "foot", FullPl: "feet", Category: UnitLength, ToBase: ratFromFrac(381, 1250)},
	{Short: "yd", Full: "yard", FullPl: "yards", Category: UnitLength, ToBase: ratFromFrac(1143, 1250)},
	{Short: "mi", Full: "mile", FullPl: "miles", Category: UnitLength, ToBase: ratFromFrac(201168, 125)},
	{Short: "au", Full: "au", FullPl: "au", Category: UnitLength, ToBase: ratFromFrac(149597870700, 1)},

	// Weight (base: grams)
	{Short: "mg", Full: "milligram", FullPl: "milligrams", Category: UnitWeight, ToBase: ratFromFrac(1, 1000)},
	{Short: "g", Full: "gram", FullPl: "grams", Category: UnitWeight, ToBase: ratFromFrac(1, 1)},
	{Short: "kg", Full: "kilogram", FullPl: "kilograms", Category: UnitWeight, ToBase: ratFromFrac(1000, 1)},
	{Short: "oz", Full: "ounce", FullPl: "ounces", Category: UnitWeight, ToBase: ratFromFrac(45359237, 1600000)},
	{Short: "lb", Full: "pound", FullPl: "pounds", Category: UnitWeight, ToBase: ratFromFrac(45359237, 100000)},

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
	{Short: "floz", Full: "floz", FullPl: "floz", Category: UnitVolume, ToBase: ratFromFrac(473176473, 16000000)},
	{Short: "cup", Full: "cup", FullPl: "cups", Category: UnitVolume, ToBase: ratFromFrac(473176473, 2000000)},
	{Short: "pt", Full: "pint", FullPl: "pints", Category: UnitVolume, ToBase: ratFromFrac(473176473, 1000000)},
	{Short: "qt", Full: "quart", FullPl: "quarts", Category: UnitVolume, ToBase: ratFromFrac(473176473, 500000)},
	{Short: "gal", Full: "gallon", FullPl: "gallons", Category: UnitVolume, ToBase: ratFromFrac(473176473, 125000)},

	// Temperature (base: kelvin)
	{Short: "K", Full: "kelvin", FullPl: "kelvin", Category: UnitTemperature, ToBase: ratFromFrac(1, 1), PreOffset: new(big.Rat)},
	{Short: "C", Full: "celsius", FullPl: "celsius", Category: UnitTemperature, ToBase: ratFromFrac(1, 1), PreOffset: ratFromFrac(27315, 100)},
	{Short: "F", Full: "fahrenheit", FullPl: "fahrenheit", Category: UnitTemperature, ToBase: ratFromFrac(5, 9), PreOffset: ratFromFrac(45967, 100)},

	// Pressure (base: Pascal)
	{Short: "Pa", Full: "pascal", FullPl: "pascals", Category: UnitPressure, ToBase: ratFromFrac(1, 1)},
	{Short: "kPa", Full: "kilopascal", FullPl: "kilopascals", Category: UnitPressure, ToBase: ratFromFrac(1000, 1)},
	{Short: "bar", Full: "bar", FullPl: "bars", Category: UnitPressure, ToBase: ratFromFrac(100000, 1)},
	{Short: "atm", Full: "atmosphere", FullPl: "atmospheres", Category: UnitPressure, ToBase: ratFromFrac(101325, 1)},
	{Short: "psi", Full: "psi", FullPl: "psi", Category: UnitPressure, ToBase: ratFromFrac(8896443230521, 1290320000)},

	// Force (base: Newton)
	{Short: "N", Full: "newton", FullPl: "newtons", Category: UnitForce, ToBase: ratFromFrac(1, 1)},
	{Short: "kN", Full: "kilonewton", FullPl: "kilonewtons", Category: UnitForce, ToBase: ratFromFrac(1000, 1)},
	{Short: "lbf", Full: "lbf", FullPl: "lbf", Category: UnitForce, ToBase: ratFromFrac(8896443230521, 2000000000000)},

	// Energy (base: Joule)
	{Short: "J", Full: "joule", FullPl: "joules", Category: UnitEnergy, ToBase: ratFromFrac(1, 1)},
	{Short: "kJ", Full: "kilojoule", FullPl: "kilojoules", Category: UnitEnergy, ToBase: ratFromFrac(1000, 1)},
	{Short: "Wh", Full: "watt-hour", FullPl: "watt-hours", Category: UnitEnergy, ToBase: ratFromFrac(3600, 1)},
	{Short: "kWh", Full: "kilowatt-hour", FullPl: "kilowatt-hours", Category: UnitEnergy, ToBase: ratFromFrac(3600000, 1)},
	{Short: "cal", Full: "calorie", FullPl: "calories", Category: UnitEnergy, ToBase: ratFromFrac(4184, 1000)},
	{Short: "kcal", Full: "kilocalorie", FullPl: "kilocalories", Category: UnitEnergy, ToBase: ratFromFrac(4184, 1)},
	{Short: "BTU", Full: "BTU", FullPl: "BTU", Category: UnitEnergy, ToBase: ratFromFrac(52752792631, 50000000)},

	// Power (base: Watt)
	{Short: "W", Full: "watt", FullPl: "watts", Category: UnitPower, ToBase: ratFromFrac(1, 1)},
	{Short: "kW", Full: "kilowatt", FullPl: "kilowatts", Category: UnitPower, ToBase: ratFromFrac(1000, 1)},
	{Short: "MW", Full: "megawatt", FullPl: "megawatts", Category: UnitPower, ToBase: ratFromFrac(1000000, 1)},
	{Short: "hp", Full: "horsepower", FullPl: "horsepower", Category: UnitPower, ToBase: ratFromFrac(37284993579113511, 50000000000000)},

	// Voltage (base: Volt)
	{Short: "mV", Full: "millivolt", FullPl: "millivolts", Category: UnitVoltage, ToBase: ratFromFrac(1, 1000)},
	{Short: "V", Full: "volt", FullPl: "volts", Category: UnitVoltage, ToBase: ratFromFrac(1, 1)},
	{Short: "kV", Full: "kilovolt", FullPl: "kilovolts", Category: UnitVoltage, ToBase: ratFromFrac(1000, 1)},

	// Current (base: Ampere)
	{Short: "mA", Full: "milliampere", FullPl: "milliamperes", Category: UnitCurrent, ToBase: ratFromFrac(1, 1000)},
	{Short: "A", Full: "ampere", FullPl: "amperes", Category: UnitCurrent, ToBase: ratFromFrac(1, 1)},

	// Resistance (base: Ohm)
	{Short: "ohm", Full: "ohm", FullPl: "ohms", Category: UnitResistance, ToBase: ratFromFrac(1, 1)},
	{Short: "kohm", Full: "kilohm", FullPl: "kilohms", Category: UnitResistance, ToBase: ratFromFrac(1000, 1)},

	// Data (base: bytes)
	{Short: "B", Full: "byte", FullPl: "bytes", Category: UnitData, ToBase: ratFromFrac(1, 1)},
	{Short: "KB", Full: "kilobyte", FullPl: "kilobytes", Category: UnitData, ToBase: ratFromFrac(1000, 1)},
	{Short: "MB", Full: "megabyte", FullPl: "megabytes", Category: UnitData, ToBase: ratFromFrac(1000000, 1)},
	{Short: "GB", Full: "gigabyte", FullPl: "gigabytes", Category: UnitData, ToBase: ratFromFrac(1000000000, 1)},
	{Short: "TB", Full: "terabyte", FullPl: "terabytes", Category: UnitData, ToBase: ratFromFrac(1000000000000, 1)},
	{Short: "KiB", Full: "kibibyte", FullPl: "kibibytes", Category: UnitData, ToBase: ratFromFrac(1024, 1)},
	{Short: "MiB", Full: "mebibyte", FullPl: "mebibytes", Category: UnitData, ToBase: ratFromFrac(1048576, 1)},
	{Short: "GiB", Full: "gibibyte", FullPl: "gibibytes", Category: UnitData, ToBase: ratFromFrac(1073741824, 1)},
	{Short: "TiB", Full: "tebibyte", FullPl: "tebibytes", Category: UnitData, ToBase: ratFromFrac(1099511627776, 1)},
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

// compoundHasOffset returns true if any unit in the compound has a non-zero PreOffset.
func (c *CompoundUnit) HasOffset() bool {
	if c == nil {
		return false
	}
	for _, u := range c.Num {
		if u.HasOffset() {
			return true
		}
	}
	for _, u := range c.Den {
		if u.HasOffset() {
			return true
		}
	}
	return false
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

package lang

import (
	"math/big"
)

// UnitCategory groups related units.
type UnitCategory int

const (
	UnitNumber UnitCategory = iota
	UnitLength
	UnitWeight
	UnitTime
	UnitTimestamp
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
	UnitCurrency
)

// Unit defines a unit with its category and conversion factor to the base unit.
type Unit struct {
	Short    string
	Full     string       // full singular name (e.g. "meter")
	FullPl   string       // full plural name (e.g. "meters")
	Category UnitCategory
	// ToBase is the conversion factor: value_in_base = (value + PreOffset) * ToBase
	// *big.Rat for physical units, int for display base (10/2/8/16).
	ToBase any
	// PreOffset is added before multiplying by ToBase.
	// *big.Rat for temperature offset, time.Location for timezone.
	// nil means no offset.
	PreOffset any
}

// HasOffset returns true if this unit uses an offset-based conversion (temperature).
func (u *Unit) HasOffset() bool {
	return u.Category == UnitTemperature
}

func ratFromFrac(num, denom int64) *big.Rat {
	return new(big.Rat).SetFrac64(num, denom)
}

// toBaseRat extracts the *big.Rat conversion factor from a Unit's ToBase field.
// Defaults to 1/1 if ToBase is nil or non-Rat.
func toBaseRat(u Unit) *big.Rat {
	if r, ok := u.ToBase.(*big.Rat); ok {
		return r
	}
	return new(big.Rat).SetInt64(1)
}

// preOffsetRat extracts the *big.Rat offset from a Unit's PreOffset field.
// Defaults to 0/1 if PreOffset is nil or non-Rat.
func preOffsetRat(u Unit) *big.Rat {
	if r, ok := u.PreOffset.(*big.Rat); ok {
		return r
	}
	return new(big.Rat)
}

var allUnits = []*Unit{
	// Length (base: meters)
	{Short: "pm", Full: "picometer", FullPl: "picometers", Category: UnitLength, ToBase: ratFromFrac(1, 1000000000000)},
	{Short: "nm", Full: "nanometer", FullPl: "nanometers", Category: UnitLength, ToBase: ratFromFrac(1, 1000000000)},
	{Short: "um", Full: "micrometer", FullPl: "micrometers", Category: UnitLength, ToBase: ratFromFrac(1, 1000000)},
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

	// Volume (base: liters)
	{Short: "mL", Full: "milliliter", FullPl: "milliliters", Category: UnitVolume, ToBase: ratFromFrac(1, 1000)},
	{Short: "L", Full: "liter", FullPl: "liters", Category: UnitVolume, ToBase: ratFromFrac(1, 1)},
	{Short: "floz", Full: "floz", FullPl: "floz", Category: UnitVolume, ToBase: ratFromFrac(473176473, 16000000000)},
	{Short: "cup", Full: "cup", FullPl: "cups", Category: UnitVolume, ToBase: ratFromFrac(473176473, 2000000000)},
	{Short: "pt", Full: "pint", FullPl: "pints", Category: UnitVolume, ToBase: ratFromFrac(473176473, 1000000000)},
	{Short: "qt", Full: "quart", FullPl: "quarts", Category: UnitVolume, ToBase: ratFromFrac(473176473, 500000000)},
	{Short: "gal", Full: "gallon", FullPl: "gallons", Category: UnitVolume, ToBase: ratFromFrac(473176473, 125000000)},

	// Temperature (base: kelvin)
	{Short: "K", Full: "kelvin", FullPl: "kelvin", Category: UnitTemperature, ToBase: ratFromFrac(1, 1)},
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
	{Short: "bit", Full: "bit", FullPl: "bits", Category: UnitData, ToBase: ratFromFrac(1, 8)},
	{Short: "kbit", Full: "kilobit", FullPl: "kilobits", Category: UnitData, ToBase: ratFromFrac(125, 1)},
	{Short: "Mbit", Full: "megabit", FullPl: "megabits", Category: UnitData, ToBase: ratFromFrac(125000, 1)},
	{Short: "Gbit", Full: "gigabit", FullPl: "gigabits", Category: UnitData, ToBase: ratFromFrac(125000000, 1)},
	{Short: "Tbit", Full: "terabit", FullPl: "terabits", Category: UnitData, ToBase: ratFromFrac(125000000000, 1)},
	{Short: "Kibit", Full: "kibibit", FullPl: "kibibits", Category: UnitData, ToBase: ratFromFrac(128, 1)},
	{Short: "Mibit", Full: "mebibit", FullPl: "mebibits", Category: UnitData, ToBase: ratFromFrac(131072, 1)},
	{Short: "Gibit", Full: "gibibit", FullPl: "gibibits", Category: UnitData, ToBase: ratFromFrac(134217728, 1)},
	{Short: "Tibit", Full: "tebibit", FullPl: "tebibits", Category: UnitData, ToBase: ratFromFrac(137438953472, 1)},
	{Short: "B", Full: "byte", FullPl: "bytes", Category: UnitData, ToBase: ratFromFrac(1, 1)},
	{Short: "KB", Full: "kilobyte", FullPl: "kilobytes", Category: UnitData, ToBase: ratFromFrac(1000, 1)},
	{Short: "MB", Full: "megabyte", FullPl: "megabytes", Category: UnitData, ToBase: ratFromFrac(1000000, 1)},
	{Short: "GB", Full: "gigabyte", FullPl: "gigabytes", Category: UnitData, ToBase: ratFromFrac(1000000000, 1)},
	{Short: "TB", Full: "terabyte", FullPl: "terabytes", Category: UnitData, ToBase: ratFromFrac(1000000000000, 1)},
	{Short: "KiB", Full: "kibibyte", FullPl: "kibibytes", Category: UnitData, ToBase: ratFromFrac(1024, 1)},
	{Short: "MiB", Full: "mebibyte", FullPl: "mebibytes", Category: UnitData, ToBase: ratFromFrac(1048576, 1)},
	{Short: "GiB", Full: "gibibyte", FullPl: "gibibytes", Category: UnitData, ToBase: ratFromFrac(1073741824, 1)},
	{Short: "TiB", Full: "tebibyte", FullPl: "tebibytes", Category: UnitData, ToBase: ratFromFrac(1099511627776, 1)},

	// Currency (base: each currency is its own base — no exchange rates)
	{Short: "USD", Full: "dollar", FullPl: "dollars", Category: UnitCurrency, ToBase: ratFromFrac(1, 1)},
	{Short: "EUR", Full: "euro", FullPl: "euros", Category: UnitCurrency, ToBase: ratFromFrac(1, 1)},
	{Short: "GBP", Category: UnitCurrency, ToBase: ratFromFrac(1, 1)},
	{Short: "JPY", Full: "yen", FullPl: "yen", Category: UnitCurrency, ToBase: ratFromFrac(1, 1)},
	{Short: "CAD", Category: UnitCurrency, ToBase: ratFromFrac(1, 1)},
	{Short: "AUD", Category: UnitCurrency, ToBase: ratFromFrac(1, 1)},
	{Short: "CHF", Category: UnitCurrency, ToBase: ratFromFrac(1, 1)},
}

// unitLookup maps short names, full singular, and full plural to unit pointers.
var unitLookup map[string]*Unit

// currencySymbols maps currency Short names to their display symbols.
var currencySymbols = map[string]string{
	"USD": "$",
	"EUR": "€",
	"GBP": "£",
	"JPY": "¥",
}

func init() {
	unitLookup = make(map[string]*Unit, len(allUnits)*3)
	for _, u := range allUnits {
		unitLookup[u.Short] = u
		if u.Full != "" {
			unitLookup[u.Full] = u
		}
		if u.FullPl != "" {
			unitLookup[u.FullPl] = u
		}
	}
	// Register currency symbol aliases
	unitLookup["$"] = unitLookup["USD"]
	unitLookup["€"] = unitLookup["EUR"]
	unitLookup["£"] = unitLookup["GBP"]
	unitLookup["¥"] = unitLookup["JPY"]
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

// numUnit is a sentinel unit for dimensionless (plain number) values.
var numUnit = Unit{Short: "", Category: UnitNumber, ToBase: ratFromFrac(1, 1)}

// tsUnit is a sentinel unit for absolute timestamps (unix seconds) with no timezone.
var tsUnit = Unit{Short: "timestamp", Category: UnitTimestamp, ToBase: ratFromFrac(1, 1)}

// Display-base sentinels: ToBase is an int indicating the display base.
var (
	decUnit = Unit{Short: "", Category: UnitNumber, ToBase: 10}
	hexUnit = Unit{Short: "", Category: UnitNumber, ToBase: 16}
	binUnit = Unit{Short: "", Category: UnitNumber, ToBase: 2}
	octUnit = Unit{Short: "", Category: UnitNumber, ToBase: 8}
)

// hmsUnit is a sentinel for hours-minutes-seconds display. The value is in seconds.
var hmsUnit = Unit{Short: "hms", Category: UnitNumber, ToBase: "hms"}

// CompoundUnit represents a compound unit like mi/gal.
// Dimensionless values use numUnit for both Num and Den.
type CompoundUnit struct {
	Num Unit // numUnit = dimensionless numerator
	Den Unit // numUnit = no denominator
}

// SimpleUnit creates a CompoundUnit from a single unit.
func SimpleUnit(u Unit) CompoundUnit {
	return CompoundUnit{Num: u, Den: numUnit}
}

// IsEmpty returns true if there are no units (both dimensionless).
func (c CompoundUnit) IsEmpty() bool {
	return c.Num.Category == UnitNumber && c.Den.Category == UnitNumber
}

// String formats the compound unit for display.
func (c CompoundUnit) String() string {
	if c.IsEmpty() {
		return ""
	}
	num := ""
	if c.Num.Category != UnitNumber {
		num = c.Num.Short
	}
	if c.Den.Category == UnitNumber {
		return num
	}
	if num == "" {
		num = "1"
	}
	return num + "/" + c.Den.Short
}

// HasOffset returns true if any unit in the compound has an offset-based conversion.
func (c CompoundUnit) HasOffset() bool {
	return c.Num.HasOffset() || c.Den.HasOffset()
}

// Compatible checks whether two compound units are compatible for add/sub.
func (c CompoundUnit) Compatible(other CompoundUnit) bool {
	if c.Num.Category != other.Num.Category {
		return false
	}
	if c.Den.Category != other.Den.Category {
		return false
	}
	return true
}

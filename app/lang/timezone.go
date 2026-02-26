package lang

import "time"

// timezoneTable maps abbreviation to fixed UTC offset in seconds.
var timezoneTable = map[string]int{
	"UTC":  0,
	"GMT":  0,
	"EST":  -5 * 3600,
	"EDT":  -4 * 3600,
	"CST":  -6 * 3600,
	"CDT":  -5 * 3600,
	"MST":  -7 * 3600,
	"MDT":  -6 * 3600,
	"PST":  -8 * 3600,
	"PDT":  -7 * 3600,
	"CET":  1 * 3600,
	"CEST": 2 * 3600,
	"IST":  5*3600 + 1800, // +5:30
	"JST":  9 * 3600,
	"AEST": 10 * 3600,
	"AEDT": 11 * 3600,
	"NZST": 12 * 3600,
	"NZDT": 13 * 3600,
}

// tzUnits maps timezone abbreviation to a Unit with PreOffset as time.Location.
var tzUnits map[string]Unit

func init() {
	tzUnits = make(map[string]Unit, len(timezoneTable))
	for name, offset := range timezoneTable {
		tzUnits[name] = Unit{
			Short:     "timestamp",
			Category:  UnitTimestamp,
			ToBase:    ratFromFrac(1, 1),
			PreOffset: *time.FixedZone(name, offset),
		}
	}
}

// LookupTZUnit returns a Unit for the given timezone abbreviation.
// Returns the zero Unit if not recognized (check Category == UnitTimestamp).
func LookupTZUnit(name string) (Unit, bool) {
	u, ok := tzUnits[name]
	return u, ok
}

// IsTimezone returns true if the given name is a known timezone abbreviation.
func IsTimezone(name string) bool {
	_, ok := timezoneTable[name]
	return ok
}

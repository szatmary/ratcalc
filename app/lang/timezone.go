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

// LookupTimezone returns a *time.Location for the given timezone abbreviation,
// or nil if not recognized.
func LookupTimezone(name string) *time.Location {
	offset, ok := timezoneTable[name]
	if !ok {
		return nil
	}
	return time.FixedZone(name, offset)
}

// IsTimezone returns true if the given name is a known timezone abbreviation.
func IsTimezone(name string) bool {
	_, ok := timezoneTable[name]
	return ok
}

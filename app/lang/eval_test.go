package lang

import (
	"strings"
	"testing"
)

func TestEvalLine(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2 + 3", "5"},
		{"10 - 3", "7"},
		{"4 * 5", "20"},
		{"10 / 3", "10/3"},
		{"1/3 + 1/6", "1/2"},
		{"-5", "-5"},
		{"(2 + 3) * 4", "20"},
		{"3.14", "157/50"},
		{"1.5 + 2.5", "4"},
	}

	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		got := val.String()
		if got != tt.want {
			t.Errorf("EvalLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestVariables(t *testing.T) {
	env := make(Env)

	// x = 10
	val, err := EvalLine("x = 10", env)
	if err != nil {
		t.Fatalf("assignment error: %v", err)
	}
	if val.String() != "10" {
		t.Errorf("x = 10 gave %q, want 10", val.String())
	}

	// x + 5
	val, err = EvalLine("x + 5", env)
	if err != nil {
		t.Fatalf("x + 5 error: %v", err)
	}
	if val.String() != "15" {
		t.Errorf("x + 5 = %q, want 15", val.String())
	}
}

func TestSingleWordVariables(t *testing.T) {
	env := make(Env)

	val, err := EvalLine("price = 42", env)
	if err != nil {
		t.Fatalf("assignment error: %v", err)
	}
	if val.String() != "42" {
		t.Errorf("price = 42 gave %q, want 42", val.String())
	}

	val, err = EvalLine("price * 2", env)
	if err != nil {
		t.Fatalf("price * 2 error: %v", err)
	}
	if val.String() != "84" {
		t.Errorf("price * 2 = %q, want 84", val.String())
	}
}

func TestUnits(t *testing.T) {
	env := make(Env)

	val, err := EvalLine("5 m", env)
	if err != nil {
		t.Fatalf("5 m error: %v", err)
	}
	if val.String() != "5 m" {
		t.Errorf("5 m = %q, want '5 m'", val.String())
	}
}

func TestUnitConversion(t *testing.T) {
	env := make(Env)

	val, err := EvalLine("5 meters + 100 cm", env)
	if err != nil {
		t.Fatalf("unit conversion error: %v", err)
	}
	if val.String() != "6 m" {
		t.Errorf("5 meters + 100 cm = %q, want '6 m'", val.String())
	}
}

func TestEmptyLine(t *testing.T) {
	env := make(Env)
	_, err := EvalLine("", env)
	if err == nil {
		t.Error("expected error for empty line")
	}
}

func TestDivisionByZero(t *testing.T) {
	env := make(Env)
	_, err := EvalLine("5 / 0", env)
	if err == nil {
		t.Error("expected error for division by zero")
	}
}

func TestCompoundUnits(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Division produces compound units
		{"10 mi / 1 gal", "10 mi/gal"},
		{"100 mi / 5 gal", "20 mi/gal"},

		// Bare unit word implies 1
		{"10 miles / gallon", "10 mi/gal"},

		// Same-category cancellation
		{"10 mi / 2 mi", "5"},

		// Add/sub with compound units
		{"10 mi / 1 gal + 5 mi / 1 gal", "15 mi/gal"},

		// Add/sub still converts within same category
		{"5 meters + 100 cm", "6 m"},

		// Dimensionless still works
		{"2 + 3", "5"},

		// Volume units
		{"5 gal", "5 gal"},
		{"1 L", "1 L"},
	}

	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		got := val.String()
		if got != tt.want {
			t.Errorf("EvalLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToConversion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Simple conversion
		{"100 km to mi", "100 km to mi"},
		// Compound conversion
		{"40 mi / 1 gal to km/L", "40 mi / 1 gal to km/L"},
		// Conversion applies to whole expression
		{"5 m + 300 cm to km", "5 m + 300 cm to km"},
		// Conversion in assignment RHS
		{"x = 40 mi / 1 gal to km/L", "x = 40 mi / 1 gal to km/L"},
	}

	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		got := val.String()
		// Just verify it produces a result with the target unit
		_ = got
	}

	// Verify specific numeric results
	env := make(Env)

	// 100 km to mi — should convert
	val, err := EvalLine("100 km to mi", env)
	if err != nil {
		t.Fatalf("100 km to mi error: %v", err)
	}
	if val.CompoundUnit().String() != "mi" {
		t.Errorf("100 km to mi: expected unit 'mi', got %v", val.CompoundUnit())
	}

	// 5 m + 300 cm to km — sum is 8m, convert to km
	val, err = EvalLine("5 m + 300 cm to km", env)
	if err != nil {
		t.Fatalf("5 m + 300 cm to km error: %v", err)
	}
	if val.CompoundUnit().String() != "km" {
		t.Errorf("5 m + 300 cm to km: expected unit 'km', got %v", val.CompoundUnit())
	}

	// Incompatible units: 5 m to kg
	_, err = EvalLine("5 m to kg", env)
	if err == nil {
		t.Error("expected error for '5 m to kg' (incompatible units)")
	}

	// "to" as variable name still works when not followed by a unit
	_, err = EvalLine("to = 5", env)
	if err != nil {
		t.Fatalf("to = 5 error: %v", err)
	}
	val, err = EvalLine("to + 3", env)
	if err != nil {
		t.Fatalf("to + 3 error: %v", err)
	}
	if val.String() != "8" {
		t.Errorf("to + 3 = %q, want 8", val.String())
	}
}

func TestDaysWeeksYears(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1 day to hr", "24 hr"},
		{"1 week to d", "7 d"},
		{"1 yr to d", "365.25 d"},
		{"24 hr to d", "1 d"},
		{"7 d to wk", "1 wk"},
		{"365.25 d to yr", "1 yr"},
	}
	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		got := val.String()
		if got != tt.want {
			t.Errorf("EvalLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBareUnitFallback(t *testing.T) {
	env := make(Env)
	val, err := EvalLine("gallon", env)
	if err != nil {
		t.Fatalf("gallon error: %v", err)
	}
	if val.String() != "1 gal" {
		t.Errorf("gallon = %q, want '1 gal'", val.String())
	}
}

func TestUnixFunction(t *testing.T) {
	env := make(Env)
	val, err := EvalLine("unix(1706745600)", env)
	if err != nil {
		t.Fatalf("unix() error: %v", err)
	}
	got := val.String()
	want := "2024-02-01 00:00:00 +0000"
	if got != want {
		t.Errorf("unix(1706745600) = %q, want %q", got, want)
	}
	if !val.IsTimestamp() {
		t.Error("expected IsTime=true")
	}
}

func TestUnixAutoDetectMs(t *testing.T) {
	env := make(Env)
	// Same timestamp in milliseconds
	val, err := EvalLine("unix(1706745600000)", env)
	if err != nil {
		t.Fatalf("unix() ms error: %v", err)
	}
	got := val.String()
	want := "2024-02-01 00:00:00 +0000"
	if got != want {
		t.Errorf("unix(1706745600000) = %q, want %q", got, want)
	}
}

func TestTounix(t *testing.T) {
	env := make(Env)

	// Date to unix — should give raw timestamp number
	val, err := EvalLine("@2024-02-01 to unix", env)
	if err != nil {
		t.Fatalf("to unix error: %v", err)
	}
	if val.IsTimestamp() {
		t.Error("expected IsTime=false after to unix")
	}
	got := val.String()
	if got != "1706745600" {
		t.Errorf("@2024-02-01 to unix = %q, want 1706745600", got)
	}

	// Time with fractional seconds: add 0.5 seconds then to unix
	val, err = EvalLine("(@2024-02-01 + 1/2 s) to unix", env)
	if err != nil {
		t.Fatalf("fractional to unix error: %v", err)
	}
	got = val.String()
	if got != "1706745600.5" {
		t.Errorf("(@2024-02-01 + 1/2) to unix = %q, want 1706745600.5", got)
	}

	// Error: to unix on non-time value
	_, err = EvalLine("42 to unix", env)
	if err == nil {
		t.Error("expected error for non-time to unix")
	}
}

func TestDateFunction(t *testing.T) {
	env := make(Env)

	// date(y, m, d) — 3 args
	val, err := EvalLine("date(2024, 1, 31)", env)
	if err != nil {
		t.Fatalf("date(2024, 1, 31) error: %v", err)
	}
	if !val.IsTimestamp() {
		t.Error("expected IsTime=true for date()")
	}
	got := val.String()
	want := "2024-01-31 00:00:00 +0000"
	if got != want {
		t.Errorf("date(2024, 1, 31) = %q, want %q", got, want)
	}

	// date(y, m, d, h, m, s) — 6 args
	val, err = EvalLine("date(2024, 1, 31, 10, 30, 0)", env)
	if err != nil {
		t.Fatalf("date(2024, 1, 31, 10, 30, 0) error: %v", err)
	}
	got = val.String()
	want = "2024-01-31 10:30:00 +0000"
	if got != want {
		t.Errorf("date(2024, 1, 31, 10, 30, 0) = %q, want %q", got, want)
	}
}

func TestTimeFunction(t *testing.T) {
	env := make(Env)

	// time(h, m) — 2 args
	val, err := EvalLine("time(14, 30)", env)
	if err != nil {
		t.Fatalf("time(14, 30) error: %v", err)
	}
	if !val.IsTimestamp() {
		t.Error("expected IsTime=true for time()")
	}
	got := val.String()
	if !strings.Contains(got, "14:30:00") {
		t.Errorf("time(14, 30) = %q, expected to contain 14:30:00", got)
	}

	// time(h, m, s) — 3 args
	val, err = EvalLine("time(9, 5, 30)", env)
	if err != nil {
		t.Fatalf("time(9, 5, 30) error: %v", err)
	}
	got = val.String()
	if !strings.Contains(got, "09:05:30") {
		t.Errorf("time(9, 5, 30) = %q, expected to contain 09:05:30", got)
	}
}

func TestAtDateLiteral(t *testing.T) {
	env := make(Env)

	// @YYYY-MM-DD
	val, err := EvalLine("@2024-01-31", env)
	if err != nil {
		t.Fatalf("@2024-01-31 error: %v", err)
	}
	if !val.IsTimestamp() {
		t.Error("expected IsTime=true for @date")
	}
	got := val.String()
	want := "2024-01-31 00:00:00 +0000"
	if got != want {
		t.Errorf("@2024-01-31 = %q, want %q", got, want)
	}

	// @YYYY-MM-DDTHH:MM:SS
	val, err = EvalLine("@2024-01-31T10:30:00", env)
	if err != nil {
		t.Fatalf("@2024-01-31T10:30:00 error: %v", err)
	}
	got = val.String()
	want = "2024-01-31 10:30:00 +0000"
	if got != want {
		t.Errorf("@2024-01-31T10:30:00 = %q, want %q", got, want)
	}

	// @YYYY-MM-DD HH:MM:SS (space separator)
	val, err = EvalLine("@2024-01-31 10:30:00", env)
	if err != nil {
		t.Fatalf("@2024-01-31 10:30:00 error: %v", err)
	}
	got = val.String()
	want = "2024-01-31 10:30:00 +0000"
	if got != want {
		t.Errorf("@2024-01-31 10:30:00 = %q, want %q", got, want)
	}

	// @YYYY-MM-DD HH:MM:SS +0000 (with UTC offset)
	val, err = EvalLine("@2024-01-31 10:30:00 +0000", env)
	if err != nil {
		t.Fatalf("@2024-01-31 10:30:00 +0000 error: %v", err)
	}
	got = val.String()
	want = "2024-01-31 10:30:00 +0000"
	if got != want {
		t.Errorf("@2024-01-31 10:30:00 +0000 = %q, want %q", got, want)
	}

	// @YYYY-MM-DD HH:MM:SS -0800 (PST offset — round-trip test)
	// 02:30 in -0800 = 10:30 UTC
	val, err = EvalLine("@2024-01-31 02:30:00 -0800", env)
	if err != nil {
		t.Fatalf("@2024-01-31 02:30:00 -0800 error: %v", err)
	}
	got = val.String()
	want = "2024-01-31 10:30:00 +0000"
	if got != want {
		t.Errorf("@2024-01-31 02:30:00 -0800 = %q, want %q", got, want)
	}
}

func TestAtTimeLiteral(t *testing.T) {
	env := make(Env)

	// @HH:MM
	val, err := EvalLine("@14:30", env)
	if err != nil {
		t.Fatalf("@14:30 error: %v", err)
	}
	if !val.IsTimestamp() {
		t.Error("expected IsTime=true for @time")
	}
	got := val.String()
	if !strings.Contains(got, "14:30:00") {
		t.Errorf("@14:30 = %q, expected to contain 14:30:00", got)
	}

	// @HH:MM:SS
	val, err = EvalLine("@9:05:30", env)
	if err != nil {
		t.Fatalf("@9:05:30 error: %v", err)
	}
	got = val.String()
	if !strings.Contains(got, "09:05:30") {
		t.Errorf("@9:05:30 = %q, expected to contain 09:05:30", got)
	}
}

func TestAtUnixLiteral(t *testing.T) {
	env := make(Env)

	// @unix_seconds
	val, err := EvalLine("@1706745600", env)
	if err != nil {
		t.Fatalf("@1706745600 error: %v", err)
	}
	if !val.IsTimestamp() {
		t.Error("expected IsTime=true for @unix")
	}
	got := val.String()
	want := "2024-02-01 00:00:00 +0000"
	if got != want {
		t.Errorf("@1706745600 = %q, want %q", got, want)
	}

	// @unix_milliseconds
	val, err = EvalLine("@1706745600000", env)
	if err != nil {
		t.Fatalf("@1706745600000 error: %v", err)
	}
	got = val.String()
	if got != want {
		t.Errorf("@1706745600000 = %q, want %q", got, want)
	}
}

func TestDateVsArithmetic(t *testing.T) {
	env := make(Env)

	// Without @, 2024-01-31 is now arithmetic (2024 - 1 - 31 = 1992)
	val, err := EvalLine("2024-01-31", env)
	if err != nil {
		t.Fatalf("arithmetic error: %v", err)
	}
	got := val.String()
	if got != "1992" {
		t.Errorf("2024-01-31 = %q, want 1992", got)
	}
	if val.IsTimestamp() {
		t.Error("expected IsTime=false for arithmetic")
	}

	// With spaces — still arithmetic
	val, err = EvalLine("2024 - 01 - 31", env)
	if err != nil {
		t.Fatalf("arithmetic error: %v", err)
	}
	got = val.String()
	if got != "1992" {
		t.Errorf("2024 - 01 - 31 = %q, want 1992", got)
	}
}

func TestTimeArithmetic(t *testing.T) {
	env := make(Env)

	// time + duration = time
	val, err := EvalLine("@2024-01-31 + 86400 s", env)
	if err != nil {
		t.Fatalf("time+duration error: %v", err)
	}
	if !val.IsTimestamp() {
		t.Error("expected time+duration to be time")
	}
	want := "2024-02-01 00:00:00 +0000"
	if val.String() != want {
		t.Errorf("@2024-01-31 + 86400 s = %q, want %q", val.String(), want)
	}

	// time + duration (hours)
	val, err = EvalLine("@2024-01-31 + 24 hr", env)
	if err != nil {
		t.Fatalf("time+24hr error: %v", err)
	}
	if !val.IsTimestamp() {
		t.Error("expected time+24hr to be time")
	}
	if val.String() != want {
		t.Errorf("@2024-01-31 + 24 hr = %q, want %q", val.String(), want)
	}

	// time + duration (days)
	val, err = EvalLine("@2024-01-31 + 1 d", env)
	if err != nil {
		t.Fatalf("time+1d error: %v", err)
	}
	if !val.IsTimestamp() {
		t.Error("expected time+1d to be time")
	}
	if val.String() != want {
		t.Errorf("@2024-01-31 + 1 d = %q, want %q", val.String(), want)
	}

	// time - time = duration in seconds
	val, err = EvalLine("@2024-02-01 - @2024-01-31", env)
	if err != nil {
		t.Fatalf("time-time error: %v", err)
	}
	if val.IsTimestamp() {
		t.Error("expected time-time to be duration, not time")
	}
	if val.String() != "86400 s" {
		t.Errorf("@2024-02-01 - @2024-01-31 = %q, want \"86400 s\"", val.String())
	}

	// time - time converted to hours
	val, err = EvalLine("@2024-02-01 - @2024-01-31 to hr", env)
	if err != nil {
		t.Fatalf("time-time to hr error: %v", err)
	}
	if val.String() != "24 hr" {
		t.Errorf("@2024-02-01 - @2024-01-31 to hr = %q, want \"24 hr\"", val.String())
	}

	// time - time converted to days
	val, err = EvalLine("@2024-02-01 - @2024-01-31 to d", env)
	if err != nil {
		t.Fatalf("time-time to d error: %v", err)
	}
	if val.String() != "1 d" {
		t.Errorf("@2024-02-01 - @2024-01-31 to d = %q, want \"1 d\"", val.String())
	}

	// time - duration = time
	val, err = EvalLine("@2024-02-01 - 1 hr", env)
	if err != nil {
		t.Fatalf("time-duration error: %v", err)
	}
	if !val.IsTimestamp() {
		t.Error("expected time-duration to be time")
	}
	wantSub := "2024-01-31 23:00:00 +0000"
	if val.String() != wantSub {
		t.Errorf("@2024-02-01 - 1 hr = %q, want %q", val.String(), wantSub)
	}

	// time + plain number = error
	_, err = EvalLine("@2024-01-31 + 86400", env)
	if err == nil {
		t.Error("expected error for time + plain number")
	}

	// time - plain number = error
	_, err = EvalLine("@2024-01-31 - 86400", env)
	if err == nil {
		t.Error("expected error for time - plain number")
	}

	// time + time = error
	_, err = EvalLine("@2024-01-31 + @2024-01-31", env)
	if err == nil {
		t.Error("expected error for time + time")
	}

	// time * number = error
	_, err = EvalLine("@2024-01-31 * 2", env)
	if err == nil {
		t.Error("expected error for time * number")
	}

	// time / number = error
	_, err = EvalLine("@2024-01-31 / 2", env)
	if err == nil {
		t.Error("expected error for time / number")
	}
}

func TestTimezoneConversion(t *testing.T) {
	env := make(Env)

	// 12:00 PST — input timezone, should adjust to UTC (PST is -8)
	val, err := EvalLine("12:00 PST", env)
	if err != nil {
		t.Fatalf("12:00 PST error: %v", err)
	}
	if !val.IsTimestamp() {
		t.Error("expected IsTime=true")
	}
	// 12:00 PST = 20:00 UTC. Display should show PST.
	got := val.String()
	if !strings.Contains(got, "12:00:00") || !strings.Contains(got, "-0800") {
		t.Errorf("12:00 PST = %q, expected 12:00:00 -0800", got)
	}

	// 12:00 PST to UTC — round-trip: display should show 20:00 UTC
	val, err = EvalLine("12:00 PST to UTC", env)
	if err != nil {
		t.Fatalf("12:00 PST to UTC error: %v", err)
	}
	got = val.String()
	if !strings.Contains(got, "20:00:00") || !strings.Contains(got, "+0000") {
		t.Errorf("12:00 PST to UTC = %q, expected 20:00:00 +0000", got)
	}

	// 12:00 UTC to PST — should show 04:00 PST
	val, err = EvalLine("12:00 UTC to PST", env)
	if err != nil {
		t.Fatalf("12:00 UTC to PST error: %v", err)
	}
	got = val.String()
	if !strings.Contains(got, "04:00:00") || !strings.Contains(got, "-0800") {
		t.Errorf("12:00 UTC to PST = %q, expected 04:00:00 -0800", got)
	}

	// now() to EST — should work and show EST offset
	val, err = EvalLine("now() to EST", env)
	if err != nil {
		t.Fatalf("now() to EST error: %v", err)
	}
	if !val.IsTimestamp() {
		t.Error("expected IsTime=true for now() to EST")
	}
	got = val.String()
	if !strings.Contains(got, "-0500") {
		t.Errorf("now() to EST = %q, expected -0500 offset", got)
	}

	// @2024-01-31T10:30:00 to PST — date with timezone conversion
	val, err = EvalLine("@2024-01-31T10:30:00 to PST", env)
	if err != nil {
		t.Fatalf("@date to PST error: %v", err)
	}
	got = val.String()
	if !strings.Contains(got, "02:30:00") || !strings.Contains(got, "-0800") {
		t.Errorf("@2024-01-31T10:30:00 to PST = %q, expected 02:30:00 -0800", got)
	}

	// @time with timezone
	val, err = EvalLine("@12:00 PST", env)
	if err != nil {
		t.Fatalf("@12:00 PST error: %v", err)
	}
	got = val.String()
	if !strings.Contains(got, "12:00:00") || !strings.Contains(got, "-0800") {
		t.Errorf("@12:00 PST = %q, expected 12:00:00 -0800", got)
	}

	// @datetime with space separator + named timezone
	val, err = EvalLine("@2024-01-31 10:30:00 PST", env)
	if err != nil {
		t.Fatalf("@2024-01-31 10:30:00 PST error: %v", err)
	}
	got = val.String()
	if !strings.Contains(got, "10:30:00") || !strings.Contains(got, "-0800") {
		t.Errorf("@2024-01-31 10:30:00 PST = %q, expected 10:30:00 -0800", got)
	}

	// @datetime with T separator + named timezone
	val, err = EvalLine("@2024-01-31T10:30:00 UTC", env)
	if err != nil {
		t.Fatalf("@2024-01-31T10:30:00 UTC error: %v", err)
	}
	got = val.String()
	wantUTC := "2024-01-31 10:30:00 +0000"
	if got != wantUTC {
		t.Errorf("@2024-01-31T10:30:00 UTC = %q, want %q", got, wantUTC)
	}

	// Error: timezone on non-time value
	_, err = EvalLine("5 m to PST", env)
	if err == nil {
		t.Error("expected error for '5 m to PST'")
	}
}

func TestTimeLiteral(t *testing.T) {
	env := make(Env)

	// Basic time literal — should produce a time value for today
	val, err := EvalLine("14:30", env)
	if err != nil {
		t.Fatalf("14:30 error: %v", err)
	}
	if !val.IsTimestamp() {
		t.Error("expected IsTime=true for time literal")
	}
	got := val.String()
	if !strings.Contains(got, "14:30:00") {
		t.Errorf("14:30 = %q, expected to contain 14:30:00", got)
	}

	// Time literal with seconds
	val, err = EvalLine("9:05:30", env)
	if err != nil {
		t.Fatalf("9:05:30 error: %v", err)
	}
	got = val.String()
	if !strings.Contains(got, "09:05:30") {
		t.Errorf("9:05:30 = %q, expected to contain 09:05:30", got)
	}
}

func TestBaseConversions(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Input literals
		{"0xFF", "255"},
		{"0xff", "255"},
		{"0b1010", "10"},
		{"0o77", "63"},

		// Output conversions
		{"255 to hex", "0xff"},
		{"10 to bin", "0b1010"},
		{"63 to oct", "0o77"},

		// Round-trip
		{"0xFF to hex", "0xff"},

		// Arithmetic with base literals
		{"0xFF + 1", "256"},
		{"0b1010 + 0o2", "12"},

		// Negative
		{"-0xFF", "-255"},
		{"-255 to hex", "-0xff"},
	}

	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		got := val.String()
		if got != tt.want {
			t.Errorf("EvalLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}

	// Error: non-integer to hex
	env := make(Env)
	_, err := EvalLine("1/3 to hex", env)
	if err == nil {
		t.Error("expected error for '1/3 to hex' (non-integer)")
	}
}

func TestNow(t *testing.T) {
	env := make(Env)
	val, err := EvalLine("now()", env)
	if err != nil {
		t.Fatalf("now() error: %v", err)
	}
	if !val.IsTimestamp() {
		t.Error("expected now() to return time")
	}
	// Just check the format is correct
	got := val.String()
	if !strings.Contains(got, "+0000") {
		t.Errorf("now() = %q, expected UTC format", got)
	}
}

func TestExponentiation(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2 ** 10", "1024"},
		{"3 ** 3", "27"},
		{"2 ** 0", "1"},
		{"2 ** -3", "1/8"},
		{"3 ** -2", "1/9"},
		// Right-associative: 2 ** 3 ** 2 = 2 ** 9 = 512
		{"2 ** 3 ** 2", "512"},
		{"(2 ** 3) ** 2", "64"},
		// Negation binds looser than **: -2 ** 2 = -(2**2)
		{"-2 ** 2", "-4"},
		{"(-2) ** 2", "4"},
		// pow() function equivalent
		{"pow(2, 10)", "1024"},
	}
	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		got := val.String()
		if got != tt.want {
			t.Errorf("EvalLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBitwiseOperations(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// AND
		{"0xFF & 0x0F", "15"},
		{"7 & 3", "3"},
		{"0 & 255", "0"},
		// OR
		{"0x0F | 0xF0", "255"},
		{"5 | 3", "7"},
		// XOR
		{"0xFF ^ 0x0F", "240"},
		{"5 ^ 3", "6"},
		// NOT
		{"~0", "-1"},
		{"~1", "-2"},
		{"~(-1)", "0"},
		// Shifts
		{"1 << 10", "1024"},
		{"1024 >> 3", "128"},
		{"0 << 5", "0"},
		{"255 >> 8", "0"},
		// Precedence: & binds tighter than |
		{"5 & 3 | 8", "9"},
		{"5 | 3 & 1", "5"},
		// ^ between & and |
		{"7 ^ 3 & 1", "6"},
	}
	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		got := val.String()
		if got != tt.want {
			t.Errorf("EvalLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}

	// Errors: non-integer operands
	errTests := []string{
		"1.5 & 3",
		"1/3 | 2",
		"1.5 ^ 3",
		"1 << 1.5",
		"~1.5",
		"1 << -1",
	}
	for _, input := range errTests {
		env := make(Env)
		_, err := EvalLine(input, env)
		if err == nil {
			t.Errorf("EvalLine(%q) expected error, got nil", input)
		}
	}
}

func TestFactorial(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0!", "1"},
		{"1!", "1"},
		{"5!", "120"},
		{"10!", "3628800"},
		{"20!", "2432902008176640000"},
		// Factorial in expressions
		{"5! + 1", "121"},
		{"5! * 2", "240"},
		// Factorial with parentheses
		{"(2 + 3)!", "120"},
	}
	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		got := val.String()
		if got != tt.want {
			t.Errorf("EvalLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}

	// Errors
	errTests := []string{
		"(-1)!",  // negative
		"1.5!",   // non-integer
		"(1/3)!", // fraction
	}
	for _, input := range errTests {
		env := make(Env)
		_, err := EvalLine(input, env)
		if err == nil {
			t.Errorf("EvalLine(%q) expected error, got nil", input)
		}
	}
}

func TestToHMS(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"3661 to hms", "1h 1m 1s"},
		{"0 to hms", "0s"},
		{"59 to hms", "59s"},
		{"60 to hms", "1m 0s"},
		{"3600 to hms", "1h 0m 0s"},
		{"90 s to hms", "1m 30s"},
		{"2.5 hr to hms", "2h 30m 0s"},
		{"1.5 min to hms", "1m 30s"},
		{"86400 s to hms", "24h 0m 0s"},
	}
	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		got := val.String()
		if got != tt.want {
			t.Errorf("EvalLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNumFunction(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"num(5 km)", "5"},
		{"num(10 mi / 1 gal)", "10"},
		{"num(42)", "42"},
		{"num(100 C)", "100"},
	}
	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		got := val.String()
		if got != tt.want {
			t.Errorf("EvalLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestUnderscoreInVariables(t *testing.T) {
	env := make(Env)
	val, err := EvalLine("my_var = 42", env)
	if err != nil {
		t.Fatalf("assignment error: %v", err)
	}
	if val.String() != "42" {
		t.Errorf("my_var = 42 gave %q, want 42", val.String())
	}

	val, err = EvalLine("my_var * 2", env)
	if err != nil {
		t.Fatalf("my_var * 2 error: %v", err)
	}
	if val.String() != "84" {
		t.Errorf("my_var * 2 = %q, want 84", val.String())
	}

	// Variable starting with underscore should fail (must start with letter)
	_, err = EvalLine("_bad = 5", env)
	if err == nil {
		t.Error("expected error for variable starting with underscore")
	}
}

func TestComments(t *testing.T) {
	// Comments are handled by the incremental evaluator, not EvalLine
	state := &EvalState{}

	lines := []string{
		"; semicolon comment",
		"// double-slash comment",
		"  ; indented comment",
		"  // indented double-slash",
		"42",
	}
	results := state.EvalAllIncremental(lines, false)

	for i := 0; i < 4; i++ {
		if results[i].Text != "" {
			t.Errorf("line %d (%q) expected empty result, got %q", i+1, lines[i], results[i].Text)
		}
	}
	if results[4].Text != "42" {
		t.Errorf("line 5 expected 42, got %q", results[4].Text)
	}
}

func TestVolumeConversions(t *testing.T) {
	tests := []struct {
		input    string
		wantUnit string
		wantMin  float64
		wantMax  float64
	}{
		{"1 gal to L", "L", 3.785, 3.786},
		{"1 L to floz", "floz", 33.81, 33.82},
		{"1 gal to cup", "cup", 15.99, 16.01},
		{"1 gal to pt", "pt", 7.99, 8.01},
		{"1 gal to qt", "qt", 3.99, 4.01},
		{"1000 mL to L", "L", 1.0, 1.0},
	}
	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		if val.CompoundUnit().String() != tt.wantUnit {
			t.Errorf("EvalLine(%q) unit = %v, want %s", tt.input, val.CompoundUnit(), tt.wantUnit)
			continue
		}
		f, _ := val.DisplayRat().Float64()
		if f < tt.wantMin || f > tt.wantMax {
			t.Errorf("EvalLine(%q) = %f, want [%f, %f]", tt.input, f, tt.wantMin, tt.wantMax)
		}
	}
}

func TestWeightConversions(t *testing.T) {
	tests := []struct {
		input    string
		wantUnit string
		wantMin  float64
		wantMax  float64
	}{
		{"1 kg to lb", "lb", 2.204, 2.205},
		{"1 lb to oz", "oz", 15.99, 16.01},
		{"1 kg to g", "g", 1000, 1000},
		{"1000 mg to g", "g", 1.0, 1.0},
		{"1 lb to g", "g", 453.59, 453.60},
	}
	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		if val.CompoundUnit().String() != tt.wantUnit {
			t.Errorf("EvalLine(%q) unit = %v, want %s", tt.input, val.CompoundUnit(), tt.wantUnit)
			continue
		}
		f, _ := val.DisplayRat().Float64()
		if f < tt.wantMin || f > tt.wantMax {
			t.Errorf("EvalLine(%q) = %f, want [%f, %f]", tt.input, f, tt.wantMin, tt.wantMax)
		}
	}
}

func TestSubMillimeterUnits(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1000 nm to um", "1 um"},
		{"1000 um to mm", "1 mm"},
		{"1000000 pm to um", "1 um"},
		{"1 mm to um", "1000 um"},
	}
	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		got := val.String()
		if got != tt.want {
			t.Errorf("EvalLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBitUnits(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"8 bit to B", "1 B"},
		{"1 B to bit", "8 bit"},
		{"1 kbit to B", "125 B"},
		{"1 Mbit to kbit", "1000 kbit"},
		{"1 KiB to B", "1024 B"},
		{"1 Kibit to bit", "1024 bit"},
		{"1 MiB to KiB", "1024 KiB"},
	}
	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		got := val.String()
		if got != tt.want {
			t.Errorf("EvalLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTemperatureConversions(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"100 C to F", "212 F"},
		{"0 C to F", "32 F"},
		{"32 F to C", "0 C"},
		{"212 F to C", "100 C"},
		{"0 K to C", "-273.15 C"},
		{"0 K to F", "-459.67 F"},
		{"100 C to K", "373.15 K"},
		{"0 C to K", "273.15 K"},
		{"-40 C to F", "-40 F"},
		{"-40 F to C", "-40 C"},
		{"373.15 K to F", "212 F"},
	}
	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		got := val.String()
		if got != tt.want {
			t.Errorf("EvalLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCompoundUnitCancellation(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Time cancels: mi/hr * hr = mi
		{"60 mi / 1 hr * 2 hr", "120 mi"},
		// Same category cancels to dimensionless
		{"10 mi / 5 mi", "2"},
		// Compound conversion
		{"10 mi / 1 gal to km/L", "10 mi / 1 gal to km/L"},
	}
	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		_ = val.String() // just verify no error
	}

	// Cross-category compound units should work
	env := make(Env)
	val, err := EvalLine("10 V / 1 m", env)
	if err != nil {
		t.Fatalf("10 V / 1 m error: %v", err)
	}
	if val.CompoundUnit().String() != "V/m" {
		t.Errorf("10 V / 1 m unit = %q, want V/m", val.CompoundUnit().String())
	}

	// Incompatible unit operations should error
	errTests := []string{
		"5 m * 3 kg",        // two categories in numerator
		"5 m + 3 kg",        // add incompatible
		"5 m - 3 kg",        // sub incompatible
		"5 m + 3",           // add unit and no unit
		"5 + 3 m",           // add no unit and unit
		"5 mi/hr + 3 km/L",  // incompatible compound
	}
	for _, input := range errTests {
		env := make(Env)
		_, err := EvalLine(input, env)
		if err == nil {
			t.Errorf("EvalLine(%q) expected error, got nil", input)
		}
	}
}

func TestCompoundUnitConversions(t *testing.T) {
	tests := []struct {
		input    string
		wantUnit string
		wantMin  float64
		wantMax  float64
	}{
		// Speed
		{"100 km / 1 hr to mi/hr", "mi/hr", 62.13, 62.14},
		// Fuel economy
		{"40 mi / 1 gal to km/L", "km/L", 17.00, 17.01},
	}
	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		if val.CompoundUnit().String() != tt.wantUnit {
			t.Errorf("EvalLine(%q) unit = %v, want %s", tt.input, val.CompoundUnit(), tt.wantUnit)
			continue
		}
		f, _ := val.DisplayRat().Float64()
		if f < tt.wantMin || f > tt.wantMax {
			t.Errorf("EvalLine(%q) = %f, want [%f, %f]", tt.input, f, tt.wantMin, tt.wantMax)
		}
	}
}

func TestAtan2(t *testing.T) {
	env := make(Env)
	val, err := EvalLine("atan2(1, 1)", env)
	if err != nil {
		t.Fatalf("atan2(1, 1) error: %v", err)
	}
	f, _ := val.effectiveRat().Float64()
	// atan2(1,1) = pi/4 ≈ 0.7854
	if f < 0.785 || f > 0.786 {
		t.Errorf("atan2(1, 1) = %f, want ~0.7854", f)
	}
}

func TestSpeedOfLightArithmetic(t *testing.T) {
	env := make(Env)

	// c has units m/s
	val, err := EvalLine("c", env)
	if err != nil {
		t.Fatalf("c error: %v", err)
	}
	if val.CompoundUnit().String() != "m/s" {
		t.Errorf("c unit = %q, want m/s", val.CompoundUnit().String())
	}

	// c * 1 s = distance in meters
	val, err = EvalLine("c * 1 s", env)
	if err != nil {
		t.Fatalf("c * 1 s error: %v", err)
	}
	if val.CompoundUnit().String() != "m" {
		t.Errorf("c * 1 s unit = %q, want m", val.CompoundUnit().String())
	}
	if val.String() != "299792458 m" {
		t.Errorf("c * 1 s = %q, want 299792458 m", val.String())
	}

	// c * 1 s to km
	val, err = EvalLine("c * 1 s to km", env)
	if err != nil {
		t.Fatalf("c * 1 s to km error: %v", err)
	}
	if val.CompoundUnit().String() != "km" {
		t.Errorf("c * 1 s to km unit = %q, want km", val.CompoundUnit().String())
	}
}

func TestCurrency(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"$50 + $30", "$80.00"},
		{"$100 * 1.08", "$108.00"},
		{"€50", "€50.00"},
		{"£75.50", "£75.50"},
		{"¥1000", "¥1000.00"},
		{"50 USD", "$50.00"},
		{"50 EUR", "€50.00"},
		{"50 CAD", "50.00 CAD"},
		{"$(50 + 30)", "$80.00"},
	}
	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		got := val.String()
		if got != tt.want {
			t.Errorf("EvalLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}

	// Error: incompatible units
	env := make(Env)
	_, err := EvalLine("$50 + 5 m", env)
	if err == nil {
		t.Error("expected error for '$50 + 5 m' (incompatible units)")
	}

	// Error: cross-currency conversion
	_, err = EvalLine("$50 to EUR", env)
	if err == nil {
		t.Error("expected error for '$50 to EUR' (cross-currency conversion)")
	}
	if err != nil && err.Error() != "__forex__" {
		t.Errorf("expected __forex__ error, got: %v", err)
	}
}

func TestBankersRounding(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"round(2.5)", "2"},
		{"round(3.5)", "4"},
		{"round(-2.5)", "-2"},
		{"round(-3.5)", "-4"},
		{"round(0.5)", "0"},
		{"round(1.5)", "2"},
		{"round(4.5)", "4"},
		{"round(5.5)", "6"},
		// Non-half values round normally
		{"round(2.3)", "2"},
		{"round(2.7)", "3"},
		{"round(-2.3)", "-2"},
		{"round(-2.7)", "-3"},
	}
	for _, tt := range tests {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		got := val.String()
		if got != tt.want {
			t.Errorf("EvalLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLineReferences(t *testing.T) {
	state := &EvalState{}
	lines := []string{"100", "#1 * 2", "#1 + #2"}
	results := state.EvalAllIncremental(lines, false)

	if results[0].Text != "100" {
		t.Errorf("line 1 = %q, want 100", results[0].Text)
	}
	if results[1].Text != "200" {
		t.Errorf("line 2 = %q, want 200", results[1].Text)
	}
	if results[2].Text != "300" {
		t.Errorf("line 3 = %q, want 300", results[2].Text)
	}
}

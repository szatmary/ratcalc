package lang

import (
	"math"
	"strings"
	"testing"
)


// TestLanguageSpecExamples tests every example from the Examples section
// of LANGUAGE.md to ensure the spec stays in sync with the implementation.
func TestLanguageSpecExamples(t *testing.T) {
	// Exact-match examples: input → expected output
	exact := []struct {
		input string
		want  string
	}{
		// Basic arithmetic
		{"2 + 3", "5"},
		{"1/3 + 1/6", "1/2"},

		// Dates and times
		{"@2024-01-31", "2024-01-31 00:00:00 +0000"},
		{"@2024-01-31T10:30:00", "2024-01-31 10:30:00 +0000"},
		{"date(2024, 1, 31)", "2024-01-31 00:00:00 +0000"},
		{"2024-01-31", "1992"},
		{"2024 - 01 - 31", "1992"},
		{"unix(1706745600)", "2024-02-01 00:00:00 +0000"},
		{"@1706745600", "2024-02-01 00:00:00 +0000"},
		{"unix(1706745600000)", "2024-02-01 00:00:00 +0000"},

		// Units
		{"5 meters + 100 cm", "6 m"},
		{"10 miles / gallon", "10 mi/gal"},
		{"100 mi / 5 gal", "20 mi/gal"},
		{"10 mi / 2 mi", "5"},

		// Time arithmetic
		{"@2024-01-31 + 86400 s", "2024-02-01 00:00:00 +0000"},
		{"@2024-01-31 + 24 hr", "2024-02-01 00:00:00 +0000"},
		{"@2024-01-31 + 1 d", "2024-02-01 00:00:00 +0000"},
		{"@2024-02-01 - 1 hr", "2024-01-31 23:00:00 +0000"},
		{"@2024-02-01 - @2024-01-31", "86400 s"},
		{"@2024-02-01 - @2024-01-31 to hr", "24 hr"},
		{"@2024-02-01 - @2024-01-31 to d", "1 d"},

		// Timezone with @ datetime
		{"@2024-01-31 10:30:00 PST", "2024-01-31 10:30:00 -0800"},
		{"@2024-01-31 02:30:00 -0800", "2024-01-31 10:30:00 +0000"},
		{"@2024-01-31T10:30:00 to PST", "2024-01-31 02:30:00 -0800"},

		// To unix
		{"@2024-02-01 to unix", "1706745600"},
		{"(@2024-02-01 + 1/2 s) to unix", "1706745600.5"},

		// Base conversions
		{"0xFF", "255"},
		{"0b1010", "10"},
		{"0o77", "63"},
		{"255 to hex", "0xff"},
		{"10 to bin", "0b1010"},
		{"63 to oct", "0o77"},
		{"0xFF + 1", "256"},
		{"0xFF + 1 to hex", "0x100"},

		// Duration conversions
		{"86400 s to hr", "24 hr"},
		{"24 hr to d", "1 d"},
		{"1 wk to d", "7 d"},

		// Single-digit month/day in @ dates
		{"@2026-2-25", "2026-02-25 00:00:00 +0000"},
		{"@2026-2-5", "2026-02-05 00:00:00 +0000"},
		{"@2026-12-5", "2026-12-05 00:00:00 +0000"},
		{"@2026-2-25T10:30:00", "2026-02-25 10:30:00 +0000"},
		{"@2026-2-25 10:30:00", "2026-02-25 10:30:00 +0000"},

		// Math functions
		{"sin(pi / 2)", "1"},
		{"cos(0)", "1"},
		{"sqrt(4)", "2"},
		{"log(100)", "2"},
		{"ln(e)", "1"},
		{"log2(8)", "3"},
		{"abs(-5)", "5"},
		{"ceil(3.2)", "4"},
		{"floor(3.8)", "3"},
		{"round(3.5)", "4"},
		{"pow(2, 10)", "1024"},
		{"mod(10, 3)", "1"},
		{"min(3, 7)", "3"},
		{"max(3, 7)", "7"},

		// Time extraction
		{"year(@2024-06-15)", "2024"},
		{"month(@2024-06-15)", "6"},
		{"day(@2024-06-15)", "15"},
		{"hour(@2024-06-15T10:30:00)", "10"},
		{"minute(@2024-06-15T10:30:00)", "30"},
		{"second(@2024-06-15T10:30:45)", "45"},

		// Constants
		{"c", "299792458 m/s"},

		// AU unit
		{"1 au to km", "1495978707/10 km"},

		// Percentage
		{"50%", "0.5"},
		{"100%", "1"},
		{"10%", "0.1"},
		{"200 * 10%", "20"},
		{"1000 * 5%", "50"},

		// Temperature
		{"100 C to F", "212 F"},
		{"0 C to K", "273.15 K"},
		{"32 F to C", "0 C"},
		{"212 F to C", "100 C"},
		{"0 K to C", "-273.15 C"},
		{"373.15 K to C", "100 C"},
		{"100 C to K", "373.15 K"},

		// Engineering units — exact conversions
		{"1 kPa to Pa", "1000 Pa"},
		{"1 bar to Pa", "100000 Pa"},
		{"1 kN to N", "1000 N"},
		{"1 kJ to J", "1000 J"},
		{"1 kWh to J", "3600000 J"},
		{"1 kcal to cal", "1000 cal"},
		{"1 kW to W", "1000 W"},
		{"1 MW to kW", "1000 kW"},
		{"1 kV to V", "1000 V"},
		{"1000 mV to V", "1 V"},
		{"1000 mA to A", "1 A"},
		{"1 kohm to ohm", "1000 ohm"},
		{"1 KB to B", "1000 B"},
		{"1 MB to KB", "1000 KB"},
		{"1 GB to MB", "1000 MB"},

		// Data — binary units
		{"1 KiB to B", "1024 B"},
		{"1 MiB to KiB", "1024 KiB"},
		{"1 GiB to MiB", "1024 MiB"},
		{"1 TiB to GiB", "1024 GiB"},
	}

	for _, tt := range exact {
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

// TestLanguageSpecVariables tests variable examples from LANGUAGE.md
// that require shared state across lines.
func TestLanguageSpecVariables(t *testing.T) {
	env := make(Env)

	lines := []struct {
		input string
		want  string
	}{
		{"x = 10", "10"},
		{"x + 5", "15"},
		{"price = 42", "42"},
		{"price * 2", "84"},
	}

	for _, tt := range lines {
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

// TestLanguageSpecTimeOfDay tests time-of-day examples that include today's
// date (which changes daily), so we check partial matches.
func TestLanguageSpecTimeOfDay(t *testing.T) {
	contains := []struct {
		input    string
		contains []string
	}{
		{"14:30", []string{"14:30:00", "+0000"}},
		{"@14:30", []string{"14:30:00", "+0000"}},
		{"time(14, 30)", []string{"14:30:00", "+0000"}},
		{"12:00 PST", []string{"12:00:00", "-0800"}},
		{"12:00 PST to UTC", []string{"20:00:00", "+0000"}},
		{"12:00 UTC to PST", []string{"04:00:00", "-0800"}},
		{"now()", []string{"+0000"}},
		{"now() to EST", []string{"-0500"}},
	}

	for _, tt := range contains {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		got := val.String()
		for _, sub := range tt.contains {
			if !strings.Contains(got, sub) {
				t.Errorf("EvalLine(%q) = %q, expected to contain %q", tt.input, got, sub)
			}
		}
	}
}

// TestLanguageSpecNowArithmetic tests now()-based arithmetic.
func TestLanguageSpecNowArithmetic(t *testing.T) {
	env := make(Env)

	// now() - @2024-01-01 → duration in seconds
	val, err := EvalLine("now() - @2024-01-01", env)
	if err != nil {
		t.Fatalf("now() - @2024-01-01 error: %v", err)
	}
	if val.IsTimestamp() {
		t.Error("expected duration (not time) from now() - @date")
	}
	if val.CompoundUnit().String() != "s" {
		t.Errorf("expected unit 's', got %v", val.CompoundUnit())
	}

	// now() to unix → positive integer
	val, err = EvalLine("now() to unix", env)
	if err != nil {
		t.Fatalf("now() to unix error: %v", err)
	}
	if val.IsTimestamp() {
		t.Error("expected IsTime=false after to unix")
	}
	if val.Sign() <= 0 {
		t.Errorf("now() to unix = %s, expected positive", val.String())
	}
}

// TestLanguageSpecUnitConversions tests approximate unit conversion results.
func TestLanguageSpecUnitConversions(t *testing.T) {
	env := make(Env)

	// 100 km to mi → unit should be mi
	val, err := EvalLine("100 km to mi", env)
	if err != nil {
		t.Fatalf("100 km to mi error: %v", err)
	}
	if val.CompoundUnit().String() != "mi" {
		t.Errorf("100 km to mi: expected unit 'mi', got %v", val.CompoundUnit())
	}

	// 40 mi / 1 gal to km/L → unit should be km/L
	val, err = EvalLine("40 mi / 1 gal to km/L", env)
	if err != nil {
		t.Fatalf("40 mi / 1 gal to km/L error: %v", err)
	}
	if val.CompoundUnit().String() != "km/L" {
		t.Errorf("40 mi / 1 gal to km/L: expected unit 'km/L', got %v", val.CompoundUnit())
	}

	// 5 m + 300 cm to km → should be 8/1000 km = 0.008 km
	val, err = EvalLine("5 m + 300 cm to km", env)
	if err != nil {
		t.Fatalf("5 m + 300 cm to km error: %v", err)
	}
	if val.CompoundUnit().String() != "km" {
		t.Errorf("5 m + 300 cm to km: expected unit 'km', got %v", val.CompoundUnit())
	}
	got := val.String()
	if !strings.Contains(got, "km") {
		t.Errorf("5 m + 300 cm to km = %q, expected km unit", got)
	}

	// 100 km/hr to mi/hr → speed conversion
	val, err = EvalLine("100 km / 1 hr to mi/hr", env)
	if err != nil {
		t.Fatalf("100 km/hr to mi/hr error: %v", err)
	}
	if val.CompoundUnit().String() != "mi/hr" {
		t.Errorf("100 km/hr to mi/hr: expected unit 'mi/hr', got %v", val.CompoundUnit())
	}
}

// TestLanguageSpecAMPM tests AM/PM time literal support.
func TestLanguageSpecAMPM(t *testing.T) {
	contains := []struct {
		input    string
		contains []string
	}{
		{"3:30 PM", []string{"15:30:00", "+0000"}},
		{"12:00 AM", []string{"00:00:00", "+0000"}},
		{"12:00 PM", []string{"12:00:00", "+0000"}},
		{"@3:30 PM", []string{"15:30:00", "+0000"}},
		{"3:30 pm", []string{"15:30:00", "+0000"}},
		{"3:30 PM PST", []string{"15:30:00", "-0800"}},
		{"11:00 AM", []string{"11:00:00", "+0000"}},
		{"12:30 PM", []string{"12:30:00", "+0000"}},
	}

	for _, tt := range contains {
		env := make(Env)
		val, err := EvalLine(tt.input, env)
		if err != nil {
			t.Errorf("EvalLine(%q) error: %v", tt.input, err)
			continue
		}
		got := val.String()
		for _, sub := range tt.contains {
			if !strings.Contains(got, sub) {
				t.Errorf("EvalLine(%q) = %q, expected to contain %q", tt.input, got, sub)
			}
		}
	}
}

// TestApproxConversions tests approximate unit conversions.
func TestApproxConversions(t *testing.T) {
	approx := []struct {
		input    string
		wantMin  float64
		wantMax  float64
		wantUnit string
	}{
		// Pressure
		{"1 atm to psi", 14.69, 14.70, "psi"},
		// Power
		{"100 W to hp", 0.134, 0.135, "hp"},
		// Data
		{"1 GB to MiB", 953.67, 953.68, "MiB"},
		// Energy
		{"1 BTU to J", 1055.0, 1055.1, "J"},
		// Force
		{"1 lbf to N", 4.44, 4.45, "N"},
	}

	for _, tt := range approx {
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

// TestFinanceFunctions tests fv and pv financial functions.
func TestFinanceFunctions(t *testing.T) {
	env := make(Env)

	// fv(0.05, 10, 1000) ≈ 12577.89
	val, err := EvalLine("fv(0.05, 10, 1000)", env)
	if err != nil {
		t.Fatalf("fv() error: %v", err)
	}
	f, _ := val.effectiveRat().Float64()
	if math.Abs(f-12577.89) > 1.0 {
		t.Errorf("fv(0.05, 10, 1000) = %f, want ~12577.89", f)
	}

	// pv(0.05, 10, 1000) ≈ 7721.73
	val, err = EvalLine("pv(0.05, 10, 1000)", env)
	if err != nil {
		t.Fatalf("pv() error: %v", err)
	}
	f, _ = val.effectiveRat().Float64()
	if math.Abs(f-7721.73) > 1.0 {
		t.Errorf("pv(0.05, 10, 1000) = %f, want ~7721.73", f)
	}
}

// TestPercentage tests percentage syntax.
func TestPercentage(t *testing.T) {
	env := make(Env)

	// Percentage in expressions
	val, err := EvalLine("200 * 10%", env)
	if err != nil {
		t.Fatalf("200 * 10%%: %v", err)
	}
	got := val.String()
	if got != "20" {
		t.Errorf("200 * 10%% = %q, want 20", got)
	}

	// Percentage with variable
	EvalLine("rate = 5%", env)
	val, err = EvalLine("1000 * rate", env)
	if err != nil {
		t.Fatalf("1000 * rate: %v", err)
	}
	got = val.String()
	if got != "50" {
		t.Errorf("1000 * rate = %q, want 50", got)
	}
}

// TestTemperatureErrors tests that temperature units in compound positions are rejected.
func TestTemperatureErrors(t *testing.T) {
	// Temperature in compound should error
	env := make(Env)
	_, err := EvalLine("5 m to C", env)
	if err == nil {
		t.Error("expected error for incompatible m to C")
	}
}

// TestLanguageSpecErrors tests examples that should produce errors.
func TestLanguageSpecErrors(t *testing.T) {
	errors := []struct {
		input string
		desc  string
	}{
		{"@2024-01-31 + 86400", "time + plain number"},
		{"@2024-01-31 - 86400", "time - plain number"},
		{"@2024-01-31 + @2024-01-31", "time + time"},
		{"@2024-01-31 * 2", "time * anything"},
		{"@2024-01-31 / 2", "time / anything"},
		{"5 m to kg", "incompatible units"},
	}

	for _, tt := range errors {
		env := make(Env)
		_, err := EvalLine(tt.input, env)
		if err == nil {
			t.Errorf("EvalLine(%q) expected error (%s), got nil", tt.input, tt.desc)
		}
	}
}

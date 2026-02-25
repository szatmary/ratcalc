package lang

import (
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
		{"Date(2024, 1, 31)", "2024-01-31 00:00:00 +0000"},
		{"2024-01-31", "1992"},
		{"2024 - 01 - 31", "1992"},
		{"Unix(1706745600)", "2024-02-01 00:00:00 +0000"},
		{"@1706745600", "2024-02-01 00:00:00 +0000"},
		{"Unix(1706745600000)", "2024-02-01 00:00:00 +0000"},

		// Units
		{"5 meters + 100 cm", "6 m"},
		{"10 miles / gallon", "10 mi/gal"},
		{"100 mi / 5 gal", "20 mi/gal"},
		{"10 mi / 2 mi", "5 mi/mi"},
		{"5 m * 3 s", "15 m*s"},

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
		{"my variable = 42", "42"},
		{"my variable * 2", "84"},
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
		{"Time(14, 30)", []string{"14:30:00", "+0000"}},
		{"12:00 PST", []string{"12:00:00", "-0800"}},
		{"12:00 PST to UTC", []string{"20:00:00", "+0000"}},
		{"12:00 UTC to PST", []string{"04:00:00", "-0800"}},
		{"Now()", []string{"+0000"}},
		{"Now() to EST", []string{"-0500"}},
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

// TestLanguageSpecNowArithmetic tests Now()-based arithmetic.
func TestLanguageSpecNowArithmetic(t *testing.T) {
	env := make(Env)

	// Now() - @2024-01-01 → duration in seconds
	val, err := EvalLine("Now() - @2024-01-01", env)
	if err != nil {
		t.Fatalf("Now() - @2024-01-01 error: %v", err)
	}
	if val.IsTime {
		t.Error("expected duration (not time) from Now() - @date")
	}
	if val.Unit == nil || val.Unit.String() != "s" {
		t.Errorf("expected unit 's', got %v", val.Unit)
	}

	// Now() to unix → positive integer
	val, err = EvalLine("Now() to unix", env)
	if err != nil {
		t.Fatalf("Now() to unix error: %v", err)
	}
	if val.IsTime {
		t.Error("expected IsTime=false after to unix")
	}
	if val.Rat.Sign() <= 0 {
		t.Errorf("Now() to unix = %s, expected positive", val.String())
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
	if val.Unit == nil || val.Unit.String() != "mi" {
		t.Errorf("100 km to mi: expected unit 'mi', got %v", val.Unit)
	}

	// 40 mi / 1 gal to km/L → unit should be km/L
	val, err = EvalLine("40 mi / 1 gal to km/L", env)
	if err != nil {
		t.Fatalf("40 mi / 1 gal to km/L error: %v", err)
	}
	if val.Unit == nil || val.Unit.String() != "km/L" {
		t.Errorf("40 mi / 1 gal to km/L: expected unit 'km/L', got %v", val.Unit)
	}

	// 5 m + 300 cm to km → should be 8/1000 km = 0.008 km
	val, err = EvalLine("5 m + 300 cm to km", env)
	if err != nil {
		t.Fatalf("5 m + 300 cm to km error: %v", err)
	}
	if val.Unit == nil || val.Unit.String() != "km" {
		t.Errorf("5 m + 300 cm to km: expected unit 'km', got %v", val.Unit)
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
	if val.Unit == nil || val.Unit.String() != "mi/hr" {
		t.Errorf("100 km/hr to mi/hr: expected unit 'mi/hr', got %v", val.Unit)
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

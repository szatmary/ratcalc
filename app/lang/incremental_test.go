package lang

import "testing"

func TestIncrementalBasicCaching(t *testing.T) {
	es := &EvalState{}

	lines := []string{"x = 10", "x + 5"}
	results := es.EvalAllIncremental(lines, false)

	if results[0].Text != "10" {
		t.Errorf("line 0: got %q, want 10", results[0].Text)
	}
	if results[1].Text != "15" {
		t.Errorf("line 1: got %q, want 15", results[1].Text)
	}

	// Re-evaluate with same lines — should use cache
	results2 := es.EvalAllIncremental(lines, false)
	if results2[0].Text != "10" || results2[1].Text != "15" {
		t.Error("cached results should match")
	}
}

func TestIncrementalDirtyPropagation(t *testing.T) {
	es := &EvalState{}

	lines := []string{"x = 10", "x + 5"}
	es.EvalAllIncremental(lines, false)

	// Change line 0
	lines2 := []string{"x = 20", "x + 5"}
	results := es.EvalAllIncremental(lines2, false)

	if results[0].Text != "20" {
		t.Errorf("line 0: got %q, want 20", results[0].Text)
	}
	if results[1].Text != "25" {
		t.Errorf("line 1: got %q, want 25 (should propagate)", results[1].Text)
	}
}

func TestIncrementalNowTick(t *testing.T) {
	es := &EvalState{}

	lines := []string{"now()"}
	results := es.EvalAllIncremental(lines, false)
	if results[0].IsErr {
		t.Fatalf("now() error: %s", results[0].Text)
	}

	// Re-eval with nowTicked=true should re-evaluate
	results2 := es.EvalAllIncremental(lines, true)
	if results2[0].IsErr {
		t.Fatalf("now() error on tick: %s", results2[0].Text)
	}
	// Both should be valid time strings (can't easily test value changed in same second)
	if results2[0].Text == "" {
		t.Error("expected non-empty result for now() after tick")
	}
}

func TestIncrementalNowTickWithTZ(t *testing.T) {
	es := &EvalState{}

	lines := []string{"now() to EST"}
	results := es.EvalAllIncremental(lines, false)
	if results[0].IsErr {
		t.Fatalf("now() to EST error: %s", results[0].Text)
	}

	// Re-eval with nowTicked=false should use cache (no re-eval)
	results2 := es.EvalAllIncremental(lines, false)
	if results2[0].Text != results[0].Text {
		t.Error("expected cached result when nowTicked=false")
	}

	// Re-eval with nowTicked=true should re-evaluate (UsesNow detected through TZExpr)
	results3 := es.EvalAllIncremental(lines, true)
	if results3[0].IsErr {
		t.Fatalf("now() to EST error on tick: %s", results3[0].Text)
	}
	if results3[0].Text == "" {
		t.Error("expected non-empty result for now() to EST after tick")
	}
}

func TestIncrementalEmptyAndComments(t *testing.T) {
	es := &EvalState{}

	lines := []string{"", "; comment", "// comment", "5 + 3"}
	results := es.EvalAllIncremental(lines, false)

	if results[0].Text != "" {
		t.Errorf("empty line should have empty result, got %q", results[0].Text)
	}
	if results[1].Text != "" {
		t.Errorf("; comment should have empty result, got %q", results[1].Text)
	}
	if results[2].Text != "" {
		t.Errorf("// comment should have empty result, got %q", results[2].Text)
	}
	if results[3].Text != "8" {
		t.Errorf("5 + 3: got %q, want 8", results[3].Text)
	}
}

func TestIncrementalLineCountChange(t *testing.T) {
	es := &EvalState{}

	lines := []string{"1 + 1"}
	results := es.EvalAllIncremental(lines, false)
	if results[0].Text != "2" {
		t.Errorf("got %q, want 2", results[0].Text)
	}

	// Add a line — triggers full reset
	lines2 := []string{"1 + 1", "3 + 4"}
	results2 := es.EvalAllIncremental(lines2, false)
	if results2[0].Text != "2" {
		t.Errorf("got %q, want 2", results2[0].Text)
	}
	if results2[1].Text != "7" {
		t.Errorf("got %q, want 7", results2[1].Text)
	}
}

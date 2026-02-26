package lang

import (
	"math/big"
	"strings"
)

// DepsInfo holds dependency information extracted from an AST node.
type DepsInfo struct {
	Vars    []string // variable names referenced (VarRef)
	UsesNow bool     // true if the expression calls Now()
	Assigns string   // non-empty if this is an assignment
}

// CachedLine holds the cached state for a single line.
type CachedLine struct {
	Text    string
	Node    Node
	Result  Value
	Err     error
	Deps    DepsInfo
	IsEmpty bool // line was blank or comment
}

// EvalResult is the result of evaluating a single line.
type EvalResult struct {
	Text  string // formatted result
	IsErr bool
}

// EvalState holds the incremental evaluation cache.
type EvalState struct {
	Lines []CachedLine
}

// CollectDeps walks an AST node to collect dependency info.
func CollectDeps(node Node) DepsInfo {
	var info DepsInfo
	collectDepsWalk(node, &info)
	return info
}

func collectDepsWalk(node Node, info *DepsInfo) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *VarRef:
		info.Vars = append(info.Vars, n.Name)
	case *BinaryExpr:
		collectDepsWalk(n.Left, info)
		collectDepsWalk(n.Right, info)
	case *UnaryExpr:
		collectDepsWalk(n.Operand, info)
	case *UnitExpr:
		collectDepsWalk(n.Expr, info)
	case *Assignment:
		info.Assigns = n.Name
		collectDepsWalk(n.Expr, info)
	case *FuncCall:
		if n.Name == "now" {
			info.UsesNow = true
		}
		for _, arg := range n.Args {
			collectDepsWalk(arg, info)
		}
	case *TZExpr:
		collectDepsWalk(n.Expr, info)
	case *PercentExpr:
		collectDepsWalk(n.Expr, info)
	case *NumberLit, *RatioLit, *TimeLit:
		// leaves — no deps
	}
}

// EvalAllIncremental evaluates lines incrementally, reusing cached results
// where possible. nowTicked indicates the 1-second timer fired.
func (es *EvalState) EvalAllIncremental(lines []string, nowTicked bool) []EvalResult {
	results := make([]EvalResult, len(lines))

	// Full reset when line count changes
	if len(lines) != len(es.Lines) {
		es.Lines = make([]CachedLine, len(lines))
		for i := range es.Lines {
			es.Lines[i].Text = "\x00" // force dirty
		}
	}

	env := make(Env)
	changedVars := make(map[string]bool)

	for i, line := range lines {
		cached := &es.Lines[i]
		trimmed := strings.TrimSpace(line)
		isEmpty := trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "//")

		// Determine if this line is dirty
		textChanged := cached.Text != line
		dirty := textChanged

		if !dirty && cached.Deps.UsesNow && nowTicked {
			dirty = true
		}

		if !dirty && !cached.IsEmpty {
			// Check if any dependency variable changed
			for _, dep := range cached.Deps.Vars {
				if changedVars[dep] {
					dirty = true
					break
				}
			}
		}

		if !dirty && !textChanged {
			// Clean — inject cached result into env and emit
			if !cached.IsEmpty && cached.Err == nil {
				if cached.Deps.Assigns != "" {
					env[cached.Deps.Assigns] = cached.Result
				}
				env[lineRef(i)] = cached.Result
			}
			if cached.IsEmpty {
				results[i] = EvalResult{}
			} else if cached.Err != nil {
				msg := cached.Err.Error()
				if msg == "" {
					results[i] = EvalResult{}
				} else {
					results[i] = EvalResult{Text: msg, IsErr: true}
				}
			} else {
				results[i] = EvalResult{Text: cached.Result.String()}
			}
			continue
		}

		// Dirty — re-evaluate
		cached.Text = line
		cached.IsEmpty = isEmpty

		if isEmpty {
			cached.Node = nil
			cached.Result = Value{}
			cached.Err = nil
			cached.Deps = DepsInfo{}
			results[i] = EvalResult{}
			continue
		}

		// Parse
		node, err := ParseLine(line)
		if err != nil {
			cached.Node = nil
			cached.Result = Value{}
			cached.Err = err
			cached.Deps = DepsInfo{}
			results[i] = EvalResult{Text: err.Error(), IsErr: true}
			continue
		}
		if node == nil {
			cached.Node = nil
			cached.Result = Value{}
			cached.Err = &EvalError{Msg: ""}
			cached.Deps = DepsInfo{}
			cached.IsEmpty = true
			results[i] = EvalResult{}
			continue
		}

		cached.Node = node
		cached.Deps = CollectDeps(node)

		// Evaluate
		val, err := Eval(node, env)
		oldResult := cached.Result
		cached.Result = val
		cached.Err = err

		if err != nil {
			msg := err.Error()
			if msg == "" {
				results[i] = EvalResult{}
			} else {
				results[i] = EvalResult{Text: msg, IsErr: true}
			}
			// If this was an assignment, mark as changed
			if cached.Deps.Assigns != "" {
				changedVars[cached.Deps.Assigns] = true
			}
			changedVars[lineRef(i)] = true
		} else {
			results[i] = EvalResult{Text: val.String()}
			if cached.Deps.Assigns != "" {
				env[cached.Deps.Assigns] = val
				if !ratEqual(oldResult.Rat, val.Rat) || oldResult.IsTime != val.IsTime {
					changedVars[cached.Deps.Assigns] = true
				}
			}
			env[lineRef(i)] = val
			if !ratEqual(oldResult.Rat, val.Rat) || oldResult.IsTime != val.IsTime {
				changedVars[lineRef(i)] = true
			}
		}
	}

	return results
}

func lineRef(i int) string {
	return "#" + strings.TrimLeft(strings.Repeat("0", 0), "0") + itoa(i+1)
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

func ratEqual(a, b *big.Rat) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Cmp(b) == 0
}

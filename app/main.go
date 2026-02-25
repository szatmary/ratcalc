package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ratcalc/app/lang"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

var (
	editorBg = color.NRGBA{R: 0x1E, G: 0x1E, B: 0x1E, A: 0xFF}
	editorFg = color.NRGBA{R: 0xD4, G: 0xD4, B: 0xD4, A: 0xFF}
)

func main() {
	go func() {
		w := new(app.Window)
		w.Option(app.Title("ratcalc"), app.Size(unit.Dp(1024), unit.Dp(768)))
		if err := run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(w *app.Window) error {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	th.Face = "Go Mono"
	th.TextSize = unit.Sp(14)

	es := NewEditorState()
	registerWebCallbacks(es, w)
	expl := explorer.NewExplorer(w)
	gutterRatio := 1.0 / 3.0 // right gutter as fraction of window width
	rightGutterWidth := 0
	var divider DragDivider

	// Load file from command line if provided
	if len(os.Args) > 1 {
		if err := es.LoadFile(os.Args[1]); err != nil {
			log.Printf("Failed to open %s: %v", os.Args[1], err)
		}
	}

	var shortcutTag = new(bool)
	var openCh <-chan FileResult
	var saveCh <-chan SaveResult

	// Incremental evaluation state
	evalState := &lang.EvalState{}
	prevLines := es.Lines()
	suppressNextChange := false
	nowTicked := false

	// 1-second ticker for Now() updates
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Channel-forward pattern for explorer compatibility
	events := make(chan event.Event)
	acks := make(chan struct{})
	go func() {
		for {
			ev := w.Event()
			events <- ev
			<-acks
			if _, ok := ev.(app.DestroyEvent); ok {
				return
			}
		}
	}()

	w.Option(app.Title(es.Title()))

	var ops op.Ops
	for {
		select {
		case <-ticker.C:
			nowTicked = true
			w.Invalidate()

		case result := <-openCh:
			openCh = nil
			if result.Err == nil {
				content := strings.ReplaceAll(string(result.Data), "\r\n", "\n")
				content = strings.ReplaceAll(content, "\r", "\n")
				es.Editor.SetText(content)
				es.Dirty = false
				w.Option(app.Title(es.Title()))
			}
			w.Invalidate()

		case result := <-saveCh:
			saveCh = nil
			if result.Err == nil {
				es.Dirty = false
				w.Option(app.Title(es.Title()))
			}
			w.Invalidate()

		case e := <-events:
			expl.ListenEvents(e)
			switch e := e.(type) {
			case app.DestroyEvent:
				acks <- struct{}{}
				return e.Err
			case app.FrameEvent:
				gtx := app.NewContext(&ops, e)

				// Compute gutter width from ratio; update ratio if user dragged
				windowW := gtx.Constraints.Max.X
				expectedWidth := int(gutterRatio * float64(windowW))
				if rightGutterWidth != 0 && rightGutterWidth != expectedWidth {
					// User dragged the divider — update ratio to match
					gutterRatio = float64(rightGutterWidth) / float64(windowW)
				}
				rightGutterWidth = int(gutterRatio * float64(windowW))

				// Handle keyboard shortcuts
				event.Op(gtx.Ops, shortcutTag)
				for {
					ev, ok := gtx.Event(
						key.Filter{Required: key.ModShortcut, Name: "O"},
						key.Filter{Required: key.ModShortcut, Name: "S"},
						key.Filter{Required: key.ModShortcut, Name: "="},
						key.Filter{Required: key.ModShortcut, Name: "-"},
						key.Filter{Required: key.ModShortcut, Name: "A"},
					)
					if !ok {
						break
					}
					if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
						switch ke.Name {
						case "O":
							if openCh == nil {
								openCh = OpenFileAsync(expl)
							}
						case "S":
							if saveCh == nil {
								if es.FilePath != "" {
									// Save directly
									go func() {
										err := es.SaveFile(es.FilePath)
										if err != nil {
											log.Printf("Save error: %v", err)
										}
										w.Invalidate()
									}()
								} else {
									// Save As
									saveCh = SaveFileAsync(expl, []byte(es.Editor.Text()), "untitled.txt")
								}
							}
						case "=": // Cmd+= (Cmd+Plus)
							if th.TextSize < unit.Sp(48) {
								th.TextSize += unit.Sp(2)
							}
						case "-": // Cmd+-
							if th.TextSize > unit.Sp(8) {
								th.TextSize -= unit.Sp(2)
							}
						case "A": // Cmd+A / Ctrl+A: select all
							es.Editor.SetCaret(es.Editor.Len(), 0)
						}
					}
				}

				// Process editor events
				textChanged := false
				for {
					ev, ok := es.Editor.Update(gtx)
					if !ok {
						break
					}
					switch ev.(type) {
					case widget.ChangeEvent:
						if suppressNextChange {
							suppressNextChange = false
							break
						}
						textChanged = true
						if !es.Dirty {
							es.Dirty = true
							w.Option(app.Title(es.Title()))
						}
					}
				}

				// Renumber #N references after event loop finishes
				if textChanged {
					newLines := es.Lines()
					delta := len(newLines) - len(prevLines)
					if delta != 0 {
						changePoint := findChangePoint(prevLines, newLines)
						if renumberLineRefs(es, changePoint, delta) {
							suppressNextChange = true
						}
					}
					prevLines = es.Lines()
				}

				// Evaluate all lines (incremental)
				lines := es.Lines()
				evalResults := evalState.EvalAllIncremental(lines, nowTicked)
				nowTicked = false
				results := make([]LineResult, len(evalResults))
				for idx, er := range evalResults {
					results[idx] = LineResult{Text: er.Text, IsErr: er.IsErr}
				}

				// Fill background
				paint.FillShape(gtx.Ops, editorBg, clip.Rect(image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)).Op())

				// Measure actual line height and editor padding
				lineHeight := MeasureLineHeight(gtx, th)
				topPad := gtx.Dp(unit.Dp(4)) // must match editor inset
				lineCount := es.LineCount()
				scrollY := 0

				// Current line highlight (caret line)
				caretLine, _ := es.Editor.CaretPos()
				topSpacerPx := gtx.Dp(unit.Dp(6))
				highlightY := topSpacerPx + topPad + caretLine*lineHeight
				highlightColor := color.NRGBA{R: 0x2A, G: 0x2D, B: 0x32, A: 0xFF}
				paint.FillShape(gtx.Ops, highlightColor,
					clip.Rect(image.Rect(0, highlightY, gtx.Constraints.Max.X, highlightY+lineHeight)).Op())

				// Layout: top padding, then left gutter | editor | right gutter
				layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return layout.Spacer{Height: unit.Dp(6)}.Layout(gtx)
					}),
					layout.Flexed(1, func(gtx C) D {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return LayoutLeftGutter(gtx, th, lineCount, scrollY, lineHeight, topPad)
					}),
					layout.Flexed(1, func(gtx C) D {
						return layoutEditor(gtx, th, es)
					}),
					layout.Rigid(func(gtx C) D {
						return divider.Layout(gtx, &rightGutterWidth, windowW)
					}),
					layout.Rigid(func(gtx C) D {
						return LayoutRightGutter(gtx, th, results, scrollY, lineHeight, topPad, rightGutterWidth)
					}),
				)
					}),
				)

				e.Frame(gtx.Ops)
			}
			acks <- struct{}{}
		}
	}
}


func layoutEditor(gtx C, th *material.Theme, es *EditorState) D {
	ed := material.Editor(th, &es.Editor, "")
	ed.Font = font.Font{Typeface: "Go Mono"}
	ed.Color = color.NRGBA{A: 0x00} // transparent text + caret (overlay draws colored text)
	ed.HintColor = color.NRGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xFF}
	ed.TextSize = th.TextSize
	ed.SelectionColor = color.NRGBA{R: 0x26, G: 0x4F, B: 0x78, A: 0xFF}

	return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx C) D {
		// 1. Editor layout (transparent text, handles input + selection)
		dims := ed.Layout(gtx)

		// 2. Colored text overlay clipped to editor bounds
		cl := clip.Rect(image.Rect(0, 0, dims.Size.X, dims.Size.Y)).Push(gtx.Ops)
		drawHighlightedText(gtx, th, es, dims)
		cl.Pop()

		return dims
	})
}

func drawHighlightedText(gtx C, th *material.Theme, es *EditorState, edDims D) {
	lines := es.Lines()

	// Measure line height and baseline using a probe label
	lineHeight, baseline := measureLineMetrics(gtx, th)
	if lineHeight <= 0 {
		return
	}
	ascent := lineHeight - baseline // distance from top of label to text baseline

	// CaretCoords().Y returns the text baseline position (adjusted for scroll).
	// Compute baseY = top of line 0 in viewport coordinates.
	caretLine, _ := es.Editor.CaretPos()
	caretPt := es.Editor.CaretCoords()
	baseY := caretPt.Y - float32(ascent) - float32(caretLine*lineHeight)

	// Draw each visible line's tokens
	for i, line := range lines {
		y := int(baseY + float32(i*lineHeight))
		if y+lineHeight < 0 || y > edDims.Size.Y {
			continue
		}
		tokens := Tokenize(line)
		x := 0
		for _, tok := range tokens {
			lbl := material.Label(th, th.TextSize, tok.Text)
			lbl.Color = TokenColor(tok.Kind)
			lbl.Font = font.Font{Typeface: "Go Mono"}
			lbl.MaxLines = 1

			off := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
			tgtx := gtx
			tgtx.Constraints.Min = image.Point{}
			tgtx.Constraints.Max = image.Pt(edDims.Size.X-x, lineHeight)
			dims := lbl.Layout(tgtx)
			off.Pop()

			x += dims.Size.X
		}
	}

	// Draw custom caret (since editor caret is also transparent)
	if gtx.Focused(&es.Editor) {
		cx := int(caretPt.X)
		cy := int(caretPt.Y)
		paint.FillShape(gtx.Ops, editorFg,
			clip.Rect(image.Rect(cx, cy-ascent, cx+2, cy+baseline)).Op())
		// Request redraw for caret visibility
		gtx.Execute(op.InvalidateCmd{})
	}
}

// measureLineMetrics returns the line height and baseline (distance from bottom
// to text baseline) for a single line of text at the theme's text size.
func measureLineMetrics(gtx C, th *material.Theme) (height, baseline int) {
	macro := op.Record(gtx.Ops)
	lbl := material.Label(th, th.TextSize, "0")
	lbl.MaxLines = 1
	probeGtx := gtx
	probeGtx.Constraints.Min = image.Point{}
	dims := lbl.Layout(probeGtx)
	macro.Stop()
	return dims.Size.Y, dims.Baseline
}

var lineRefRe = regexp.MustCompile(`#(\d+)`)

// findChangePoint compares old and new lines, returns the 0-indexed position
// where they first differ.
func findChangePoint(oldLines, newLines []string) int {
	n := len(oldLines)
	if len(newLines) < n {
		n = len(newLines)
	}
	for i := 0; i < n; i++ {
		if oldLines[i] != newLines[i] {
			return i
		}
	}
	return n
}

// renumberLineRefs adjusts #N references when lines are inserted or removed.
// changePoint is the 0-indexed line where old and new text first differ.
// References #N where N > changePoint (1-indexed > 0-indexed) are shifted by delta.
// Returns true if text was modified.
func renumberLineRefs(es *EditorState, changePoint, delta int) bool {
	lines := es.Lines()
	changed := false

	for i := 0; i < len(lines); i++ {
		newLine := lineRefRe.ReplaceAllStringFunc(lines[i], func(match string) string {
			m := lineRefRe.FindStringSubmatch(match)
			n, _ := strconv.Atoi(m[1])
			if n > changePoint {
				n += delta
				if n < 1 {
					n = 1
				}
				changed = true
				return fmt.Sprintf("#%d", n)
			}
			return match
		})
		lines[i] = newLine
	}

	if changed {
		newText := strings.Join(lines, "\n")
		// Restore caret by line/col since rune offsets shift with renumbering
		caretLine, caretCol := es.Editor.CaretPos()
		es.Editor.SetText(newText)
		// Convert line/col back to rune offset
		offset := 0
		for i := 0; i < caretLine && i < len(lines); i++ {
			offset += len([]rune(lines[i])) + 1 // +1 for newline
		}
		offset += caretCol
		es.Editor.SetCaret(offset, offset)
	}
	return changed
}

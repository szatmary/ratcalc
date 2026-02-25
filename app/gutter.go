package main

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

var (
	gutterBg       = color.NRGBA{R: 0x1E, G: 0x1E, B: 0x1E, A: 0xFF}
	gutterFg       = color.NRGBA{R: 0x85, G: 0x85, B: 0x85, A: 0xFF}
	gutterDivider  = color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xFF}
	gutterWidth    = unit.Dp(50)
	rightGutterBg  = color.NRGBA{R: 0x1E, G: 0x1E, B: 0x1E, A: 0xFF}
	resultColor    = color.NRGBA{R: 0x4E, G: 0xC9, B: 0xB0, A: 0xFF} // teal
	resultErrColor = color.NRGBA{R: 0xF4, G: 0x47, B: 0x47, A: 0xFF} // red
)

// LineResult holds the evaluation result for a single line.
type LineResult struct {
	Text  string // formatted result or error message
	IsErr bool
}

// MeasureLineHeight measures the actual rendered line height for the given theme.
func MeasureLineHeight(gtx layout.Context, th *material.Theme) int {
	// Record ops so the probe label doesn't actually render
	macro := op.Record(gtx.Ops)
	lbl := material.Label(th, th.TextSize, "0")
	lbl.MaxLines = 1
	probeGtx := gtx
	probeGtx.Constraints.Min = image.Point{}
	dims := lbl.Layout(probeGtx)
	macro.Stop()
	if dims.Size.Y > 0 {
		return dims.Size.Y
	}
	return gtx.Sp(th.TextSize)
}

// LayoutLeftGutter renders line numbers in a fixed-width column.
// scrollY is the vertical scroll offset in pixels, topPad is extra top padding to match the editor.
func LayoutLeftGutter(gtx layout.Context, th *material.Theme, lineCount int, scrollY int, lineHeight int, topPad int) layout.Dimensions {
	width := gtx.Dp(gutterWidth)
	height := gtx.Constraints.Max.Y

	// Background
	rect := image.Rect(0, 0, width, height)
	paint.FillShape(gtx.Ops, gutterBg, clip.Rect(rect).Op())

	if lineHeight <= 0 {
		lineHeight = 16
	}

	firstLine := scrollY / lineHeight
	if firstLine < 0 {
		firstLine = 0
	}
	visibleLines := height/lineHeight + 2
	lastLine := firstLine + visibleLines
	if lastLine > lineCount {
		lastLine = lineCount
	}

	// Determine number width for formatting
	digits := len(fmt.Sprintf("%d", lineCount))
	if digits < 2 {
		digits = 2
	}
	fmtStr := fmt.Sprintf("%%%dd", digits)

	// Render visible line numbers
	for i := firstLine; i < lastLine; i++ {
		yOffset := topPad + i*lineHeight - scrollY
		if yOffset+lineHeight < 0 || yOffset > height {
			continue
		}

		lineLabel := material.Label(th, th.TextSize, fmt.Sprintf(fmtStr, i+1))
		lineLabel.Color = gutterFg
		lineLabel.Alignment = text.End
		lineLabel.MaxLines = 1

		// Position with offset, clip to line bounds, then render
		off := op.Offset(image.Pt(0, yOffset)).Push(gtx.Ops)
		cl := clip.Rect(image.Rect(0, 0, width-gtx.Dp(4), lineHeight)).Push(gtx.Ops)
		labelGtx := gtx
		labelGtx.Constraints = layout.Exact(image.Pt(width-gtx.Dp(4), lineHeight))
		lineLabel.Layout(labelGtx)
		cl.Pop()
		off.Pop()
	}

	// Draw faint divider line on the right edge
	dividerX := width - 1
	paint.FillShape(gtx.Ops, gutterDivider, clip.Rect(image.Rect(dividerX, 0, dividerX+1, height)).Op())

	return layout.Dimensions{Size: image.Pt(width, height)}
}

// LayoutRightGutter renders the right gutter with evaluation results.
// widthPx is the gutter width in pixels.
func LayoutRightGutter(gtx layout.Context, th *material.Theme, results []LineResult, scrollY int, lineHeight int, topPad int, widthPx int) layout.Dimensions {
	width := widthPx
	height := gtx.Constraints.Max.Y

	if lineHeight <= 0 {
		lineHeight = 16
	}

	lineCount := len(results)
	firstLine := scrollY / lineHeight
	if firstLine < 0 {
		firstLine = 0
	}
	visibleLines := height/lineHeight + 2
	lastLine := firstLine + visibleLines
	if lastLine > lineCount {
		lastLine = lineCount
	}

	for i := firstLine; i < lastLine; i++ {
		r := results[i]
		if r.Text == "" {
			continue
		}

		yOffset := topPad + i*lineHeight - scrollY
		if yOffset+lineHeight < 0 || yOffset > height {
			continue
		}

		c := resultColor
		if r.IsErr {
			c = resultErrColor
		}

		lbl := material.Label(th, th.TextSize, r.Text)
		lbl.Color = c
		lbl.Alignment = text.Start
		lbl.MaxLines = 1

		off := op.Offset(image.Pt(gtx.Dp(8), yOffset)).Push(gtx.Ops)
		cl := clip.Rect(image.Rect(0, 0, width-gtx.Dp(16), lineHeight)).Push(gtx.Ops)
		labelGtx := gtx
		labelGtx.Constraints = layout.Exact(image.Pt(width-gtx.Dp(16), lineHeight))
		lbl.Layout(labelGtx)
		cl.Pop()
		off.Pop()
	}

	return layout.Dimensions{Size: image.Pt(width, height)}
}

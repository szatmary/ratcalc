package main

import (
	"image"
	"image/color"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// DragDivider is a draggable vertical divider that resizes the right gutter.
type DragDivider struct {
	dragging   bool
	startX     float32
	startWidth int
	tag        bool
}

var dividerColor = color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xFF}
var dividerHoverColor = color.NRGBA{R: 0x60, G: 0x60, B: 0x60, A: 0xFF}

const dividerWidthPx = 6
const minGutterWidth = 80
const maxGutterWidth = 600

// Layout renders the drag handle and processes pointer events.
// It mutates *width based on drag deltas.
func (d *DragDivider) Layout(gtx layout.Context, width *int) layout.Dimensions {
	height := gtx.Constraints.Max.Y

	// Process pointer events
	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: &d.tag,
			Kinds:  pointer.Press | pointer.Drag | pointer.Release,
		})
		if !ok {
			break
		}
		if pe, ok := ev.(pointer.Event); ok {
			switch pe.Kind {
			case pointer.Press:
				d.dragging = true
				d.startX = pe.Position.X
				d.startWidth = *width
			case pointer.Drag:
				if d.dragging {
					delta := int(pe.Position.X - d.startX)
					newWidth := d.startWidth - delta
					if newWidth < minGutterWidth {
						newWidth = minGutterWidth
					}
					if newWidth > maxGutterWidth {
						newWidth = maxGutterWidth
					}
					*width = newWidth
				}
			case pointer.Release, pointer.Cancel:
				d.dragging = false
			}
		}
	}

	// Draw the divider
	c := dividerColor
	if d.dragging {
		c = dividerHoverColor
	}
	rect := image.Rect(0, 0, dividerWidthPx, height)
	paint.FillShape(gtx.Ops, c, clip.Rect(rect).Op())

	// Register pointer input area and set resize cursor
	area := clip.Rect(rect).Push(gtx.Ops)
	event.Op(gtx.Ops, &d.tag)
	pointer.CursorColResize.Add(gtx.Ops)
	area.Pop()

	return layout.Dimensions{Size: image.Pt(dividerWidthPx, height)}
}

//go:build js && wasm

package main

import (
	"syscall/js"

	"gioui.org/app"
)

func registerWebCallbacks(es *EditorState, w *app.Window) {
	js.Global().Set("getEditorText", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return es.Editor.Text()
	}))
	js.Global().Set("setEditorText", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			es.Editor.SetText(args[0].String())
			w.Invalidate()
		}
		return nil
	}))

	// Load initial text from URL parameter (decoded by JS before WASM started)
	initialText := js.Global().Get("_initialText")
	if !initialText.IsUndefined() && !initialText.IsNull() && initialText.String() != "" {
		es.Editor.SetText(initialText.String())
	}
}

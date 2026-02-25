//go:build !(js && wasm)

package main

import "gioui.org/app"

func registerWebCallbacks(_ *EditorState, _ *app.Window) {}

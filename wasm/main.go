package main

import (
	"ratcalc/app/lang"
	"strings"
	"syscall/js"

	"github.com/klauspost/compress/zstd"
)

var (
	evalState  = &lang.EvalState{}
	editorText string
	zstdEnc, _ = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	zstdDec, _ = zstd.NewReader(nil)
)

func main() {
	// Register evaluate function
	js.Global().Set("evaluate", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 2 {
			return nil
		}
		text := args[0].String()
		nowTicked := args[1].Bool()
		editorText = text

		lines := strings.Split(text, "\n")
		results := evalState.EvalAllIncremental(lines, nowTicked)

		arr := js.Global().Get("Array").New(len(results))
		for i, r := range results {
			obj := js.Global().Get("Object").New()
			obj.Set("text", r.Text)
			obj.Set("isErr", r.IsErr)
			arr.SetIndex(i, obj)
		}
		return arr
	}))

	// Register getEditorText for share link
	js.Global().Set("getEditorText", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return editorText
	}))

	// Register setEditorText for share link restore
	js.Global().Set("setEditorText", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			editorText = args[0].String()
			// Update textarea via JS callback
			ta := js.Global().Get("document").Call("getElementById", "editor")
			if !ta.IsUndefined() && !ta.IsNull() {
				ta.Set("value", editorText)
				ta.Call("dispatchEvent", js.Global().Get("Event").New("input"))
			}
		}
		return nil
	}))

	// Register zstd compress/decompress for share links
	js.Global().Set("zstdCompress", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return nil
		}
		src := []byte(args[0].String())
		dst := zstdEnc.EncodeAll(src, nil)
		arr := js.Global().Get("Uint8Array").New(len(dst))
		js.CopyBytesToJS(arr, dst)
		return arr
	}))

	js.Global().Set("zstdDecompress", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return nil
		}
		src := make([]byte, args[0].Get("length").Int())
		js.CopyBytesToGo(src, args[0])
		dst, err := zstdDec.DecodeAll(src, nil)
		if err != nil {
			return js.Null()
		}
		return string(dst)
	}))

	// Signal that WASM is ready
	js.Global().Set("_wasmReady", true)
	onReady := js.Global().Get("_onWasmReady")
	if !onReady.IsUndefined() && !onReady.IsNull() {
		onReady.Invoke()
	}

	// Block forever
	select {}
}

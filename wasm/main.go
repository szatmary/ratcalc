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

	// Register setMaxDisplayLen for dynamic gutter width
	js.Global().Set("setMaxDisplayLen", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 1 {
			lang.MaxDisplayLen = args[0].Int()
		}
		return nil
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

	// Register tokenize function for syntax highlighting
	js.Global().Set("tokenize", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return nil
		}
		text := args[0].String()
		lines := strings.Split(text, "\n")
		result := js.Global().Get("Array").New(len(lines))
		for i, line := range lines {
			tokens := lang.Lex(line)
			lineArr := js.Global().Get("Array").New(len(tokens))
			for j, t := range tokens {
				obj := js.Global().Get("Object").New()
				obj.Set("type", int(t.Type))
				obj.Set("pos", t.Pos)
				obj.Set("lit", t.Literal)
				lineArr.SetIndex(j, obj)
			}
			result.SetIndex(i, lineArr)
		}
		return result
	}))

	// Register isUnit function for syntax highlighting
	js.Global().Set("isUnit", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return false
		}
		return lang.LookupUnit(args[0].String()) != nil
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

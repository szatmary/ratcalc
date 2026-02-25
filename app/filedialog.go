package main

import (
	"io"

	"gioui.org/x/explorer"
)

// FileResult holds the result of a file open operation.
type FileResult struct {
	Data []byte
	Path string
	Err  error
}

// SaveResult holds the result of a file save operation.
type SaveResult struct {
	Err error
}

// OpenFileAsync triggers a file-open dialog in a goroutine.
// The result is sent on the returned channel.
func OpenFileAsync(expl *explorer.Explorer) <-chan FileResult {
	ch := make(chan FileResult, 1)
	go func() {
		file, err := expl.ChooseFile()
		if err != nil {
			ch <- FileResult{Err: err}
			return
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		ch <- FileResult{Data: data, Err: err}
	}()
	return ch
}

// SaveFileAsync triggers a file-save dialog in a goroutine.
// The result is sent on the returned channel.
func SaveFileAsync(expl *explorer.Explorer, content []byte, defaultName string) <-chan SaveResult {
	ch := make(chan SaveResult, 1)
	go func() {
		w, err := expl.CreateFile(defaultName)
		if err != nil {
			ch <- SaveResult{Err: err}
			return
		}
		_, err = w.Write(content)
		if closeErr := w.Close(); err == nil {
			err = closeErr
		}
		ch <- SaveResult{Err: err}
	}()
	return ch
}

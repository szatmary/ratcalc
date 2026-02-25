package main

import (
	"os"
	"strings"

	"gioui.org/widget"
)

// EditorState holds the state for the text editor.
type EditorState struct {
	Editor   widget.Editor
	FilePath string
	Dirty    bool
}

// NewEditorState creates a new editor with default settings.
func NewEditorState() *EditorState {
	es := &EditorState{}
	es.Editor.SingleLine = false
	es.Editor.Submit = false
	return es
}

// Lines returns the text buffer as a slice of lines.
func (es *EditorState) Lines() []string {
	t := es.Editor.Text()
	if t == "" {
		return []string{""}
	}
	return strings.Split(t, "\n")
}

// LineCount returns the number of lines in the buffer.
func (es *EditorState) LineCount() int {
	return len(es.Lines())
}

// LoadFile reads a file and sets the editor content.
func (es *EditorState) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	// Normalize line endings
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	es.Editor.SetText(content)
	es.FilePath = path
	es.Dirty = false
	return nil
}

// SaveFile writes the editor content to the given path.
func (es *EditorState) SaveFile(path string) error {
	err := os.WriteFile(path, []byte(es.Editor.Text()), 0644)
	if err != nil {
		return err
	}
	es.FilePath = path
	es.Dirty = false
	return nil
}

// Title returns a window title string showing filename and dirty state.
func (es *EditorState) Title() string {
	name := "untitled"
	if es.FilePath != "" {
		// Use just the filename, not the full path
		parts := strings.Split(es.FilePath, "/")
		name = parts[len(parts)-1]
	}
	if es.Dirty {
		return "* " + name + " — ratcalc"
	}
	return name + " — ratcalc"
}

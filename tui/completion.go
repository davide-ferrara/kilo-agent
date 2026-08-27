package tui

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

type CompletionKind int

const (
	CompletionNone CompletionKind = iota
	CompletionCommand
	CompletionFile
)

type CompletionItem struct {
	Value       string
	Description string
}

type Completion struct {
	Kind     CompletionKind
	Start    int
	Items    []CompletionItem
	Selected int
}

var commands = []CompletionItem{
	{"/help", "show commands and keyboard shortcuts"},
	{"/clear", "clear the conversation"},
	{"/tokens", "show detailed context usage"},
	{"/quit", "exit Kilo Agent"},
}

func discoverFiles(root string, limit int) []string {
	files := make([]string, 0, limit)
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." {
			return nil
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "node_modules" || entry.Name() == "vendor") {
			return filepath.SkipDir
		}
		if !entry.IsDir() {
			files = append(files, filepath.ToSlash(rel))
			if len(files) >= limit {
				return fs.SkipAll
			}
		}
		return nil
	})
	sort.Strings(files)
	return files
}

func updateCompletion(model *Model) {
	text := model.Editor.Text
	cursor := clamp(model.Editor.Cursor, 0, len(text))
	start := cursor
	for start > 0 && !isTokenBoundary(text[start-1]) {
		start--
	}
	token := string(text[start:cursor])
	completion := Completion{Start: start}

	if strings.HasPrefix(token, "/") && start == 0 {
		completion.Kind = CompletionCommand
		for _, command := range commands {
			if strings.HasPrefix(command.Value, token) {
				completion.Items = append(completion.Items, command)
			}
		}
	} else if strings.HasPrefix(token, "@") {
		completion.Kind = CompletionFile
		query := strings.ToLower(strings.TrimPrefix(token, "@"))
		for _, name := range model.Files {
			if fuzzyContains(strings.ToLower(name), query) {
				value := "@" + name
				if strings.ContainsRune(name, ' ') {
					value = "@{" + name + "}"
				}
				completion.Items = append(completion.Items, CompletionItem{value, "file"})
				if len(completion.Items) == 8 {
					break
				}
			}
		}
	}
	if len(completion.Items) > 0 {
		model.Completion = completion
	} else {
		model.Completion = Completion{}
	}
}

func applyCompletion(model *Model) bool {
	if len(model.Completion.Items) == 0 {
		return false
	}
	item := model.Completion.Items[clamp(model.Completion.Selected, 0, len(model.Completion.Items)-1)]
	start := model.Completion.Start
	tail := append([]rune(nil), model.Editor.Text[model.Editor.Cursor:]...)
	replacement := append([]rune(nil), model.Editor.Text[:start]...)
	replacement = append(replacement, []rune(item.Value)...)
	model.Editor.Text = append(replacement, tail...)
	model.Editor.Cursor = start + len([]rune(item.Value))
	if model.Completion.Kind == CompletionFile {
		model.Editor.Insert(" ")
	}
	model.Completion = Completion{}
	return true
}

func fuzzyContains(value, query string) bool {
	if query == "" || strings.Contains(value, query) {
		return true
	}
	queryRunes := []rune(query)
	index := 0
	for _, r := range value {
		if index < len(queryRunes) && r == queryRunes[index] {
			index++
		}
	}
	return index == len(queryRunes)
}

func isTokenBoundary(r rune) bool { return r == ' ' || r == '\t' || r == '\n' }

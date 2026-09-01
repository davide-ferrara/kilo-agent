package tui

import "testing"

func TestEditorInsertionAndReadlineBindings(t *testing.T) {
	editor := Editor{Text: []rune("hello world"), Cursor: 5}
	editor.Insert(", brave")
	if got := editor.String(); got != "hello, brave world" {
		t.Fatalf("insert = %q", got)
	}
	editor.Cursor = len(editor.Text)
	editor.DeleteWord()
	if got := editor.String(); got != "hello, brave " {
		t.Fatalf("delete word = %q", got)
	}
	editor.Insert(string(editor.KillRing))
	if got := editor.String(); got != "hello, brave world" {
		t.Fatalf("yank = %q", got)
	}
}

func TestEditorMovesVerticallyAcrossWrappedInput(t *testing.T) {
	editor := Editor{Text: []rune("abcdefghij"), Cursor: 9, PreferredColumn: -1}
	editor.MoveVertical(-1, 5)
	if editor.Cursor != 4 {
		t.Fatalf("cursor after up = %d, want 4", editor.Cursor)
	}
	editor.MoveVertical(1, 5)
	if editor.Cursor != 9 {
		t.Fatalf("cursor after down = %d, want 9", editor.Cursor)
	}
}

func TestEditorMovesVerticallyAcrossExplicitLines(t *testing.T) {
	editor := Editor{Text: []rune("abcd\nxy\n12345"), Cursor: 12, PreferredColumn: -1}
	editor.MoveVertical(-1, 20)
	if editor.Cursor != 7 {
		t.Fatalf("cursor after first up = %d, want end of short middle line at 7", editor.Cursor)
	}
	editor.MoveVertical(-1, 20)
	if editor.Cursor != 4 {
		t.Fatalf("cursor after second up = %d, want column 4 on first line", editor.Cursor)
	}
	editor.MoveVertical(1, 20)
	if editor.Cursor != 7 {
		t.Fatalf("cursor after down = %d, want preferred column clamped to 7", editor.Cursor)
	}
}

func TestEditorExactWrapFollowedByNewlineIsOneBoundary(t *testing.T) {
	editor := Editor{Text: []rune("abcde\nxy"), Cursor: 8, PreferredColumn: -1}
	editor.MoveVertical(-1, 5)
	if editor.Cursor != 2 {
		t.Fatalf("cursor after up = %d, want column 2 on wrapped line", editor.Cursor)
	}
	editor.MoveVertical(1, 5)
	if editor.Cursor != 8 {
		t.Fatalf("cursor after down = %d, want original position 8", editor.Cursor)
	}
}

func TestEditorMutationResetsPreferredColumn(t *testing.T) {
	editor := Editor{Text: []rune("long\nx"), Cursor: 4, PreferredColumn: 3}
	editor.Insert("!")
	if editor.PreferredColumn != -1 {
		t.Fatalf("preferred column after insert = %d, want reset", editor.PreferredColumn)
	}
}

func TestEditorKillBindingsOnlyChangeCurrentLine(t *testing.T) {
	editor := Editor{Text: []rune("one\ntwo\nthree"), Cursor: 6, PreferredColumn: -1}
	editor.KillToStart()
	if got := editor.String(); got != "one\no\nthree" {
		t.Fatalf("kill to start = %q", got)
	}
	editor.Cursor = 4
	editor.KillToEnd()
	if got := editor.String(); got != "one\n\nthree" {
		t.Fatalf("kill to end = %q", got)
	}
}

func TestEditorHistoryRestoresDraft(t *testing.T) {
	editor := Editor{History: []string{"old"}, HistoryIndex: 1, PreferredColumn: -1}
	editor.Insert("draft")
	editor.Recall(-1)
	if got := editor.String(); got != "old" {
		t.Fatalf("history up = %q", got)
	}
	editor.Recall(1)
	if got := editor.String(); got != "draft" {
		t.Fatalf("history down = %q, want original draft", got)
	}
}

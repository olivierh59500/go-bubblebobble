package app

import (
	"path/filepath"
	"testing"

	"bubblebobble/internal/game"
	"bubblebobble/internal/save"
	"bubblebobble/resources"
)

func TestEditorSaveReloadPlayAndUndo(t *testing.T) {
	levels, err := game.LoadCampaign(resources.Files)
	if err != nil {
		t.Fatal(err)
	}
	s, err := save.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := &App{store: s, levels: levels, page: pageEditor, editor: editor{level: levels[0].Clone()}}
	a.editorAction("new")
	if len(a.editor.level.Enemies) != 0 {
		t.Fatal("new level kept template enemies")
	}
	a.editor.paint(game.Cell{Col: 10, Row: 23}, 5)
	a.editor.remember()
	a.editor.paint(game.Cell{Col: 10, Row: 20}, 0)
	if a.editor.level.Tiles[20][10] == 0 {
		t.Fatal("painting a block failed")
	}
	a.editorAction("undo")
	if a.editor.level.Tiles[20][10] != 0 {
		t.Fatal("undo did not restore tiles")
	}
	a.editorAction("save-level")
	reloaded, err := save.Open(filepath.Dir(s.Path))
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Data.Custom == nil || len(reloaded.Data.Custom.Enemies) != 1 {
		t.Fatal("custom level not saved")
	}
	a.editorAction("test-level")
	if a.page != pagePlay || a.match == nil || !a.match.Custom {
		t.Fatal("custom play test did not start")
	}
	a.activate("leave")
	if a.page != pageEditor || a.store.Profile().Played != 0 {
		t.Fatal("custom run modified campaign statistics")
	}
	a.editor.level.Tiles[20][10] = 12
	if reloaded.Data.Custom.Tiles[20][10] != 0 {
		t.Fatal("editor mutated saved level")
	}
}

func TestEndRunRecordedOnlyOnce(t *testing.T) {
	levels, err := game.LoadCampaign(resources.Files)
	if err != nil {
		t.Fatal(err)
	}
	s, err := save.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := &App{store: s, levels: levels}
	a.start(false)
	a.match.Score = 1200
	a.record(false)
	a.record(false)
	if s.Profile().Played != 1 || s.Profile().HighScore != 1200 {
		t.Fatal("run recorded incorrectly")
	}
	a.activate("restart")
	if s.Profile().Played != 1 || a.match.Score != 0 || a.match.Players[0].Lives != 3 {
		t.Fatal("restart did not start a clean run")
	}
}

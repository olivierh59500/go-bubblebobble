package save

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"bubblebobble/internal/game"
	"bubblebobble/resources"
)

func TestProfilesSettingsAndCustomLevelRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Create(" alice "); err != nil {
		t.Fatal(err)
	}
	s.Profile().Avatar = "blue"
	s.Data.Settings.Music = 0
	s.Data.Settings.Sound = 100
	s.Data.Settings.Fullscreen = true
	levels, err := game.LoadCampaign(resources.Files)
	if err != nil {
		t.Fatal(err)
	}
	l := levels[16].Clone()
	s.Data.Custom = &l
	if err = s.Record(true, 42000, 25); err != nil {
		t.Fatal(err)
	}
	if err = s.Record(false, 120, 7); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.Data, reloaded.Data) {
		t.Fatal("saved data did not survive reload")
	}
	p := reloaded.Profile()
	if p.Played != 2 || p.Won != 1 || p.Lost != 1 || p.XP != 100 || p.HighScore != 42000 || p.BestRound != 25 {
		t.Fatalf("invalid statistics: %+v", p)
	}
	if reloaded.Leaderboard()[0].Name != "ALICE" {
		t.Fatal("leaderboard not sorted")
	}
	files, _ := filepath.Glob(filepath.Join(dir, ".save-*"))
	if len(files) != 0 {
		t.Fatal("atomic write left temporary files")
	}
}

func TestProfileValidation(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "../escape", "a/other", "name longer than sixteen", "bad\nname"} {
		if err := s.Create(name); err == nil {
			t.Fatalf("accepted profile name %q", name)
		}
	}
	if err = s.Create("player1"); err == nil {
		t.Fatal("accepted duplicate profile")
	}
}

func TestCorruptOrFutureSaveIsNotOverwritten(t *testing.T) {
	for _, data := range []string{"{broken", `{"version":99}`, `{"version":1,"profiles":[],"active":8}`} {
		dir := t.TempDir()
		path := filepath.Join(dir, "save.json")
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(dir); err == nil {
			t.Fatal("accepted corrupt save")
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != data {
			t.Fatal("corrupt save was overwritten")
		}
	}
}

func TestWriteErrorIsReported(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "blocked"), []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	s.Path = filepath.Join(dir, "blocked", "save.json")
	if err = s.Write(); err == nil {
		t.Fatal("write failure was hidden")
	}
}

package main

import "testing"

func TestLoadLevelOne(t *testing.T) {
	level, err := loadLevel(1)
	if err != nil {
		t.Fatalf("load level 1: %v", err)
	}
	if !level.LoadedFromDisk {
		t.Fatal("level 1 should be loaded from disk")
	}
	if level.Tiles.get(0, 0, false) != tileSolid {
		t.Fatal("expected level 1 to contain top border tiles")
	}
	if level.ShadeBottom.A == 0 || level.ShadeRight.A == 0 {
		t.Fatal("expected shade colors to be parsed")
	}
}

func TestMissingLevelUsesEmptyLayout(t *testing.T) {
	level, err := loadLevel(26)
	if err != nil {
		t.Fatalf("missing level should not be fatal: %v", err)
	}
	if !level.MissingLevelFile {
		t.Fatal("expected level 26 to be marked missing in the current asset set")
	}
}

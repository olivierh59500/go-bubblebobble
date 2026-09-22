package game

import (
	"encoding/json"
	"fmt"
	"io/fs"
)

const (
	Columns   = 32
	Rows      = 26
	Tile      = 24
	Width     = Columns * Tile
	Height    = Rows * Tile
	TPS       = 60
	ActorSize = 44
)

type Cell struct {
	Col int `json:"col"`
	Row int `json:"row"`
}

type EnemyKind string

const (
	ZenChan EnemyKind = "zenchan"
	Monsta  EnemyKind = "monsta"
	Mighta  EnemyKind = "mighta"
	Pulpul  EnemyKind = "pulpul"
	Banebou EnemyKind = "banebou"
	Invader EnemyKind = "invader"
)

var EnemyKinds = []EnemyKind{ZenChan, Monsta, Mighta, Pulpul, Banebou, Invader}

type EnemySpawn struct {
	Kind EnemyKind `json:"kind"`
	Cell
}

type Element string

const (
	Water   Element = "water"
	Thunder Element = "thunder"
	Fire    Element = "fire"
)

type Level struct {
	Number       int          `json:"number"`
	Tiles        [][]int      `json:"tiles"`
	Enemies      []EnemySpawn `json:"enemies"`
	Player       Cell         `json:"player"`
	PowerSpawn   Cell         `json:"power_spawn"`
	FoodSpawn    Cell         `json:"food_spawn"`
	Special      Element      `json:"special"`
	SpecialCount int          `json:"special_count"`
}

func LoadCampaign(files fs.FS) ([]Level, error) {
	data, err := fs.ReadFile(files, "levels.json")
	if err != nil {
		return nil, err
	}
	var levels []Level
	if err := json.Unmarshal(data, &levels); err != nil {
		return nil, err
	}
	if len(levels) != 25 {
		return nil, fmt.Errorf("campaign has %d levels, want 25", len(levels))
	}
	for i, level := range levels {
		if level.Number != i+1 {
			return nil, fmt.Errorf("unexpected level number %d", level.Number)
		}
		if err := level.Validate(); err != nil {
			return nil, fmt.Errorf("level %d: %w", i+1, err)
		}
	}
	return levels, nil
}

func (l Level) Validate() error {
	if l.Number < 1 || l.Number > 25 {
		return fmt.Errorf("tile theme must be between 1 and 25")
	}
	if len(l.Tiles) != Rows {
		return fmt.Errorf("expected %d rows", Rows)
	}
	for row, tiles := range l.Tiles {
		if len(tiles) != Columns {
			return fmt.Errorf("row %d must contain %d tiles", row, Columns)
		}
		for col, tile := range tiles {
			if tile < 0 || tile > 25 {
				return fmt.Errorf("invalid tile at %d,%d", col, row)
			}
			if (col < 2 || col >= Columns-2) && tile == 0 {
				return fmt.Errorf("side walls must remain closed")
			}
		}
	}
	if len(l.Enemies) == 0 || len(l.Enemies) > 32 {
		return fmt.Errorf("place between 1 and 32 enemies")
	}
	for _, e := range l.Enemies {
		valid := false
		for _, kind := range EnemyKinds {
			valid = valid || e.Kind == kind
		}
		if !valid {
			return fmt.Errorf("unknown enemy %q", e.Kind)
		}
		if !validCell(e.Cell) {
			return fmt.Errorf("enemy is outside the arena")
		}
	}
	if !validCell(l.Player) || !validCell(l.PowerSpawn) || !validCell(l.FoodSpawn) {
		return fmt.Errorf("spawn point is outside the arena")
	}
	if l.Special != "" && l.Special != Water && l.Special != Thunder && l.Special != Fire {
		return fmt.Errorf("unknown special bubble %q", l.Special)
	}
	if l.SpecialCount < 0 || l.SpecialCount > 100 {
		return fmt.Errorf("invalid special bubble count")
	}
	space := false
	for row := 1; row < Rows-2; row++ {
		for col := 2; col < Columns-3; col++ {
			if l.Tiles[row][col] == 0 && l.Tiles[row][col+1] == 0 && l.Tiles[row+1][col] == 0 && l.Tiles[row+1][col+1] == 0 {
				space = true
			}
		}
	}
	if !space {
		return fmt.Errorf("leave room for the player and enemies")
	}
	return nil
}

func validCell(c Cell) bool { return c.Col >= 2 && c.Col < Columns-2 && c.Row >= 1 && c.Row < Rows }

func (l Level) Clone() Level {
	c := l
	c.Tiles = make([][]int, len(l.Tiles))
	for i := range l.Tiles {
		c.Tiles[i] = append([]int(nil), l.Tiles[i]...)
	}
	c.Enemies = append([]EnemySpawn(nil), l.Enemies...)
	return c
}

func (l Level) Solid(col, row int) bool {
	if col < 2 || col >= Columns-2 {
		return true
	}
	if row < 0 || row >= Rows {
		return false
	}
	return l.Tiles[row][col] != 0
}

// Cell coordinates name the platform underneath an actor's feet.
func (c Cell) Position(size float64) (float64, float64) {
	return min(float64(c.Col*Tile), Width-2*Tile-size), max(float64(Tile), float64(c.Row*Tile)-size)
}

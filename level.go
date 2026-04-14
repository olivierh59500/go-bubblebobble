package main

import (
	"fmt"
	"image/color"
	"os"
	"strconv"
	"strings"
)

type levelTileType int

const (
	tileNone levelTileType = iota
	tileSolid
	tileAirflowUp
	tileAirflowDown
	tileAirflowRight
	tileAirflowLeft
	tileEnemyCanLeft
	tileEnemyCanRight
	tileEnemyGhostLeft
	tileEnemyGhostRight
	tileEnemyPurpleLeft
	tileEnemyPurpleRight
	tileEnemyPigLeft
	tileEnemyPigRight
	tileEnemyMushroomLeft
	tileEnemyMushroomRight
	tileEnemySnowmanLeft
	tileEnemySnowmanRight
	tileEnemyPotatoLeft
	tileEnemyPotatoRight
	tileEnemyWitchLeft
	tileEnemyWitchRight
	tileItemSoup
	tileItemMeal
	tileItemDoor
	tileItemShoe
	tileItemPotion
	tileItemFlamingo
)

type tilemap struct {
	Data [levelTileCount]levelTileType
}

func (t *tilemap) get(x, y int, checked bool) levelTileType {
	if checked && outOfLevelRange(x, y) {
		return tileNone
	}
	return t.Data[tileIndex(x, y)]
}

func (t *tilemap) set(x, y int, value levelTileType) {
	t.Data[tileIndex(x, y)] = value
}

func (t *tilemap) setIndex(index int, value levelTileType) {
	t.Data[index] = value
}

func (t *tilemap) isEmpty(x, y int) bool {
	return t.get(x, y, false) == tileNone
}

type levelLayout struct {
	Tiles            tilemap
	Airflow          tilemap
	Enemies          tilemap
	Items            tilemap
	ShadeRight       color.RGBA
	ShadeBottom      color.RGBA
	ContainsBoss     bool
	LoadedFromDisk   bool
	SourceLevel      int
	MissingLevelFile bool
}

type tiledMap struct {
	Layers   []tiledLayer `json:"layers"`
	Tilesets []struct {
		FirstGID int    `json:"firstgid"`
		Source   string `json:"source"`
	} `json:"tilesets"`
}

type tiledLayer struct {
	Name       string          `json:"name"`
	Data       []int           `json:"data"`
	Properties []tiledProperty `json:"properties"`
}

type tiledProperty struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

func loadLevel(level int) (levelLayout, error) {
	path := fmt.Sprintf("res/levels/Level%s.json", levelFileNumber(level))
	layout := levelLayout{
		ShadeRight:  colorBlack,
		ShadeBottom: colorBlack,
		SourceLevel: level,
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			layout.MissingLevelFile = true
			return layout, nil
		}
		return layout, err
	}
	var data tiledMap
	if err := loadJSON(path, &data); err != nil {
		return layout, err
	}
	layout.LoadedFromDisk = true

	const (
		tsBlock = iota
		tsAirflow
		tsEnemy
		tsItems
		tsCount
	)
	var gids [tsCount]int
	for _, tileset := range data.Tilesets {
		switch {
		case strings.Contains(tileset.Source, "LevelBlocks.tsx"):
			gids[tsBlock] = tileset.FirstGID
		case strings.Contains(tileset.Source, "AirflowTileset.tsx"):
			gids[tsAirflow] = tileset.FirstGID
		case strings.Contains(tileset.Source, "EnemyTilesDirected.tsx"):
			gids[tsEnemy] = tileset.FirstGID
		case strings.Contains(tileset.Source, "SpecialItems.tsx"):
			gids[tsItems] = tileset.FirstGID
		}
	}

	for _, layer := range data.Layers {
		switch layer.Name {
		case "Tiles":
			layout.ShadeBottom = propertyColor(layer, "ShadeBottom", colorBlack)
			layout.ShadeRight = propertyColor(layer, "ShadeRight", colorBlack)
			layout.ContainsBoss = propertyBool(layer, "SpawnBoss")
			for i, value := range normalizedLayerData(layer.Data) {
				if value > 0 {
					layout.Tiles.setIndex(i, tileSolid)
				}
			}
			if propertyBool(layer, "FlipAlongXAxis") {
				for x := 0; x < levelWidth/2; x++ {
					for y := 0; y < levelHeight; y++ {
						if layout.Tiles.get(x, y, false) == tileNone {
							layout.Tiles.set(levelWidth-1-x, y, tileNone)
						} else {
							layout.Tiles.set(levelWidth-1-x, y, tileSolid)
						}
					}
				}
			}
		case "Airflow":
			for i, value := range normalizedLayerData(layer.Data) {
				if value == 0 {
					layout.Airflow.setIndex(i, tileNone)
				} else {
					layout.Airflow.setIndex(i, levelTileType(value-gids[tsAirflow]+int(tileAirflowUp)))
				}
			}
			if propertyBool(layer, "FlipAlongXAxis") {
				for x := 0; x < levelWidth/2; x++ {
					for y := 0; y < levelHeight; y++ {
						layout.Airflow.set(levelWidth-1-x, y, flipTileAlongX(layout.Airflow.get(x, y, false)))
					}
				}
			}
		case "Enemies":
			for i, value := range normalizedLayerData(layer.Data) {
				if value == 0 {
					layout.Enemies.setIndex(i, tileNone)
				} else {
					layout.Enemies.setIndex(i, levelTileType(value-gids[tsEnemy]+int(tileEnemyCanLeft)))
				}
			}
			if propertyBool(layer, "FlipAlongXAxis") {
				for x := 0; x < levelWidth/2; x++ {
					for y := 0; y < levelHeight; y++ {
						targetX := levelWidth - 1 - x - 1
						if targetX >= 0 {
							layout.Enemies.set(targetX, y, flipTileAlongX(layout.Enemies.get(x, y, false)))
						}
					}
				}
			}
		case "Items":
			for i, value := range normalizedLayerData(layer.Data) {
				if value == 0 {
					layout.Items.setIndex(i, tileNone)
				} else {
					layout.Items.setIndex(i, levelTileType(value-gids[tsItems]+int(tileItemSoup)))
				}
			}
		}
	}

	return layout, nil
}

func normalizedLayerData(data []int) []int {
	out := make([]int, levelTileCount)
	copy(out, data)
	return out
}

func propertyBool(layer tiledLayer, name string) bool {
	for _, property := range layer.Properties {
		if property.Name == name {
			switch value := property.Value.(type) {
			case bool:
				return value
			case string:
				return value == "true"
			}
		}
	}
	return false
}

func propertyColor(layer tiledLayer, name string, fallback color.RGBA) color.RGBA {
	for _, property := range layer.Properties {
		if property.Name == name {
			if value, ok := property.Value.(string); ok {
				return parseTiledColor(value, fallback)
			}
		}
	}
	return fallback
}

func parseTiledColor(value string, fallback color.RGBA) color.RGBA {
	if len(value) != 9 || value[0] != '#' {
		return fallback
	}
	a, errA := strconv.ParseUint(value[1:3], 16, 8)
	r, errR := strconv.ParseUint(value[3:5], 16, 8)
	g, errG := strconv.ParseUint(value[5:7], 16, 8)
	b, errB := strconv.ParseUint(value[7:9], 16, 8)
	if errA != nil || errR != nil || errG != nil || errB != nil {
		return fallback
	}
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
}

func flipTileAlongX(tile levelTileType) levelTileType {
	switch tile {
	case tileAirflowRight:
		return tileAirflowLeft
	case tileAirflowLeft:
		return tileAirflowRight
	case tileEnemyCanLeft:
		return tileEnemyCanRight
	case tileEnemyCanRight:
		return tileEnemyCanLeft
	case tileEnemyGhostLeft:
		return tileEnemyGhostRight
	case tileEnemyGhostRight:
		return tileEnemyGhostLeft
	case tileEnemyPurpleLeft:
		return tileEnemyPurpleRight
	case tileEnemyPurpleRight:
		return tileEnemyPurpleLeft
	case tileEnemyPigLeft:
		return tileEnemyPigRight
	case tileEnemyPigRight:
		return tileEnemyPigLeft
	case tileEnemyMushroomLeft:
		return tileEnemyMushroomRight
	case tileEnemyMushroomRight:
		return tileEnemyMushroomLeft
	case tileEnemySnowmanLeft:
		return tileEnemySnowmanRight
	case tileEnemySnowmanRight:
		return tileEnemySnowmanLeft
	case tileEnemyPotatoLeft:
		return tileEnemyPotatoRight
	case tileEnemyPotatoRight:
		return tileEnemyPotatoLeft
	case tileEnemyWitchLeft:
		return tileEnemyWitchRight
	case tileEnemyWitchRight:
		return tileEnemyWitchLeft
	default:
		return tile
	}
}

func tileIndex(x, y int) int {
	return x + y*levelWidth
}

func outOfLevelRange(x, y int) bool {
	return x < 0 || x >= levelWidth || y < 0 || y >= levelHeight
}

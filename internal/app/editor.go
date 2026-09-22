package app

import (
	"fmt"
	"image"
	"strings"

	"bubblebobble/internal/game"
	"github.com/hajimehoshi/ebiten/v2"
)

const editorX, editorY, editorTile = 24, 146, 16

type editor struct {
	level    game.Level
	tool     int
	template int
	dragging bool
	lastCell game.Cell
	history  []game.Level
}

var editorTools = []string{"BLOCK", "ERASER", "PLAYER", "FOOD SPAWN", "BONUS SPAWN", "ZEN-CHAN", "MONSTA", "MIGHTA", "PULPUL", "BANEBOU", "INVADER"}

func (a *App) editorButtons() []button {
	var b []button
	add := func(label, action string, x, y, w int) {
		b = append(b, button{label, action, image.Rect(x, y, x+w, y+30)})
	}
	for i, name := range editorTools {
		if a.editor.tool == i {
			name = "> " + name
		}
		add(name, fmt.Sprintf("tool:%d", i), 554, 146+i*35, 190)
	}
	add("<", "template-prev", 24, 91, 40)
	add(fmt.Sprintf("COPY ROUND %02d", a.editor.template+1), "template", 76, 91, 248)
	add(">", "template-next", 336, 91, 40)
	add("NEW", "new", 388, 91, 68)
	add("UNDO", "undo", 468, 91, 68)
	add("SAVE", "save-level", 24, 594, 116)
	add("LOAD", "load-level", 156, 594, 116)
	add("PLAY TEST", "test-level", 288, 594, 148)
	add("BACK", "menu", 452, 594, 84)
	element := strings.ToUpper(string(a.editor.level.Special))
	if element == "" {
		element = "NONE"
	}
	add("BUBBLES: "+element, "element", 24, 642, 304)
	add(fmt.Sprintf("TILE: %02d", a.editor.level.Number), "theme", 344, 642, 192)
	return b
}

func (a *App) updateEditor() {
	a.updateButtons()
	if a.page != pageEditor || a.ignoreTouch {
		return
	}
	if pressed(ebiten.KeyZ) && held(ebiten.KeyControl, ebiten.KeyMeta) {
		a.editorAction("undo")
	}
	if pressed(ebiten.KeyS) && held(ebiten.KeyControl, ebiten.KeyMeta) {
		a.editorAction("save-level")
	}
	if pressed(ebiten.KeyF5) {
		a.start(true)
		return
	}
	pt := a.scenePoint()
	x, y := pt.X, pt.Y
	left, right := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft), ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	if len(a.touchIDs) > 0 {
		x, y = ebiten.TouchPosition(a.touchIDs[0])
		x -= a.controls.SceneX
		left = true
	}
	if !left && !right {
		a.editor.dragging = false
		return
	}
	inside := image.Pt(x, y).In(image.Rect(editorX, editorY, editorX+game.Columns*editorTile, editorY+game.Rows*editorTile))
	if !inside {
		return
	}
	c := game.Cell{Col: (x - editorX) / editorTile, Row: (y - editorY) / editorTile}
	if c.Col < 2 || c.Col >= game.Columns-2 {
		return
	}
	if !a.editor.dragging {
		a.editor.remember()
		a.editor.lastCell = game.Cell{Col: -1, Row: -1}
		a.editor.dragging = true
	}
	if c == a.editor.lastCell {
		return
	}
	a.editor.lastCell = c
	tool := a.editor.tool
	if right {
		tool = 1
	}
	a.editor.paint(c, tool)
}

func (e *editor) remember() {
	e.history = append(e.history, e.level.Clone())
	if len(e.history) > 50 {
		e.history = e.history[len(e.history)-50:]
	}
}

func (e *editor) paint(c game.Cell, tool int) {
	switch tool {
	case 0:
		e.level.Tiles[c.Row][c.Col] = e.level.Number
	case 1:
		e.level.Tiles[c.Row][c.Col] = 0
		es := e.level.Enemies[:0]
		for _, enemy := range e.level.Enemies {
			if enemy.Col != c.Col || enemy.Row != c.Row+2 {
				es = append(es, enemy)
			}
		}
		e.level.Enemies = es
	default:
		// The preview selects an actor's head; level data stores its feet.
		c.Row = min(game.Rows-1, max(2, c.Row+2))
		switch tool {
		case 2:
			e.level.Player = c
		case 3:
			e.level.FoodSpawn = c
		case 4:
			e.level.PowerSpawn = c
		default:
			if tool-5 < 0 || tool-5 >= len(game.EnemyKinds) || len(e.level.Enemies) >= 32 {
				return
			}
			for i := range e.level.Enemies {
				if e.level.Enemies[i].Cell == c {
					e.level.Enemies[i].Kind = game.EnemyKinds[tool-5]
					return
				}
			}
			e.level.Enemies = append(e.level.Enemies, game.EnemySpawn{Kind: game.EnemyKinds[tool-5], Cell: c})
		}
	}
}

func (a *App) editorAction(action string) {
	e := &a.editor
	switch action {
	case "template-prev":
		e.template = (e.template + 24) % 25
	case "template-next":
		e.template = (e.template + 1) % 25
	case "template":
		e.remember()
		e.level = a.levels[e.template].Clone()
		a.message("ROUND COPIED")
	case "new":
		e.remember()
		e.level = a.levels[0].Clone()
		e.level.Enemies = nil
		e.level.Special = ""
		e.level.SpecialCount = 0
		for row := range e.level.Tiles {
			for col := 2; col < game.Columns-2; col++ {
				if row > 0 && row < game.Rows-1 {
					e.level.Tiles[row][col] = 0
				}
			}
		}
	case "undo":
		if len(e.history) > 0 {
			e.level = e.history[len(e.history)-1]
			e.history = e.history[:len(e.history)-1]
		}
	case "save-level":
		if err := e.level.Validate(); err != nil {
			a.message(err.Error())
			return
		}
		l := e.level.Clone()
		a.store.Data.Custom = &l
		if err := a.store.Write(); err != nil {
			a.message("SAVE FAILED: " + err.Error())
		} else {
			a.message("CUSTOM LEVEL SAVED")
		}
	case "load-level":
		if a.store.Data.Custom == nil {
			a.message("NO CUSTOM LEVEL SAVED YET")
			return
		}
		e.remember()
		e.level = a.store.Data.Custom.Clone()
		a.message("CUSTOM LEVEL LOADED")
	case "test-level":
		a.start(true)
	case "element":
		e.remember()
		switch e.level.Special {
		case "":
			e.level.Special = game.Water
		case game.Water:
			e.level.Special = game.Thunder
		case game.Thunder:
			e.level.Special = game.Fire
		default:
			e.level.Special = ""
		}
		e.level.SpecialCount = 16
		if e.level.Special == "" {
			e.level.SpecialCount = 0
		}
	case "theme":
		e.remember()
		e.level.Number = e.level.Number%25 + 1
		for row := range e.level.Tiles {
			for col, tile := range e.level.Tiles[row] {
				if tile != 0 {
					e.level.Tiles[row][col] = e.level.Number
				}
			}
		}
	default:
		if strings.HasPrefix(action, "tool:") {
			fmt.Sscanf(action, "tool:%d", &e.tool)
		}
	}
}

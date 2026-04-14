package main

import (
	"image/color"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type playState int

const (
	stateTitle playState = iota
	stateGameplay
)

type modifiers struct {
	SpeedUp  bool
	FireRate bool
	RangeUp  bool
}

type game struct {
	assets *assets
	audio  *audioManager
	rng    *rand.Rand
	state  playState

	levelNumber int
	level       *levelLayout

	nextID int

	dragons     []dragon
	enemies     []enemy
	bubbles     []bubble
	projectiles []projectile
	tumbles     []tumble
	items       []item
	floatTexts  []floatingText

	titleSprites []simpleSprite
	titleTexts   []uiText
	titleEntity  int
	timeCounter  int

	score              int
	playerCount        int
	modifiers          modifiers
	waitingForNext     bool
	nextLevelCountdown int
}

func newGame() (*game, error) {
	assets, err := loadAssets()
	if err != nil {
		return nil, err
	}
	audio, err := loadAudioManager()
	if err != nil {
		return nil, err
	}
	g := &game{
		assets:      assets,
		audio:       audio,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
		state:       stateTitle,
		playerCount: 1,
	}
	g.setupTitle()
	return g, nil
}

func (g *game) nextEntityID() int {
	g.nextID++
	return g.nextID
}

func (g *game) setupTitle() {
	g.titleSprites = nil
	g.titleTexts = nil
	g.titleSprites = append(g.titleSprites,
		simpleSprite{Pos: position{X: 80, Y: 0, Dir: -1}, Sprite: g.assets.mustSprite("TitleScreen-Title"), Color: colorWhite, ScaleX: 2, ScaleY: 2},
		simpleSprite{Pos: position{X: 0, Y: 288, Dir: 1}, Sprite: g.assets.mustSprite("TitleScreen-Dragon"), Color: colorWhite, ScaleX: 2, ScaleY: 2},
		simpleSprite{Pos: position{X: 390, Y: 288, Dir: -1}, Sprite: g.assets.mustSprite("TitleScreen-Dragon"), Color: colorDarkBlue, ScaleX: 2, ScaleY: 2},
		simpleSprite{Pos: position{X: 56, Y: 122, Dir: -1}, Sprite: g.assets.mustSprite("Bubble-Green-Idle-1"), Color: colorWhite, ScaleX: 2, ScaleY: 2},
		simpleSprite{Pos: position{X: -5, Y: 250, Dir: -1}, Sprite: g.assets.mustSprite("Bubble-Green-Idle-1"), Color: colorWhite, ScaleX: 2, ScaleY: 2},
		simpleSprite{Pos: position{X: 418, Y: 38, Dir: -1}, Sprite: g.assets.mustSprite("Bubble-Green-Idle-1"), Color: colorWhite, ScaleX: 2, ScaleY: 2},
		simpleSprite{Pos: position{X: 486, Y: 78, Dir: -1}, Sprite: g.assets.mustSprite("Bubble-Green-Idle-1"), Color: colorWhite, ScaleX: 2, ScaleY: 2},
		simpleSprite{Pos: position{X: 456, Y: 222, Dir: -1}, Sprite: g.assets.mustSprite("Bubble-Green-Idle-1"), Color: colorWhite, ScaleX: 2, ScaleY: 2},
	)
	g.titleEntity = 0
	g.titleTexts = append(g.titleTexts,
		uiText{Pos: vec2i{X: 324, Y: 688}, Text: "Press any key", Color: colorWhite, FontSize: 32, Spacing: -4},
		uiText{Pos: vec2i{X: 736, Y: 4}, Text: "Fan Remake\nusing Ebiten", Color: colorBlue, FontSize: 24},
	)
}

func (g *game) startGameplay() {
	g.state = stateGameplay
	g.score = 0
	g.levelNumber = 1
	g.modifiers = modifiers{}
	g.startLevel()
}

func (g *game) startLevel() {
	layout, err := loadLevel(g.levelNumber)
	if err != nil {
		layout = levelLayout{ShadeRight: colorBlack, ShadeBottom: colorBlack, SourceLevel: g.levelNumber}
	}
	g.level = &layout
	g.dragons = nil
	g.enemies = nil
	g.bubbles = nil
	g.projectiles = nil
	g.tumbles = nil
	g.items = nil
	g.floatTexts = nil
	g.waitingForNext = false
	g.nextLevelCountdown = 0

	for x := 0; x < levelWidth; x++ {
		for y := 0; y < levelHeight; y++ {
			if !layout.Enemies.isEmpty(x, y) {
				tile := layout.Enemies.get(x, y, false)
				g.createEnemy(bpSize(x+2, 0), bpSize(y, 0), enemyTypeFromTile(tile), directionFromEnemyTile(tile), true)
			}
			if !layout.Items.isEmpty(x, y) {
				tile := layout.Items.get(x, y, false)
				if tile != tileNone {
					g.createItem(vec2i{X: bpSize(x+2, 0), Y: bpSize(y, 0)}, itemTypeFromTile(tile))
				}
			}
		}
	}
	if layout.ContainsBoss {
		g.createEnemy(bpSize(10, 0), bpSize(12, 0), enemyBoss, -1, false)
	}
	g.createDragon(dragonGreen, false)
	if g.playerCount > 1 {
		g.createDragon(dragonBlue, false)
	}
}

func (g *game) createDragon(color dragonColor, invincible bool) {
	d := dragon{
		ID:    g.nextEntityID(),
		Pos:   dragonStartingPosition(color),
		Color: color,
		Actor: walkingActor{FallSpeed: unitsPerBlock / 8, JumpSpeed: 3 * unitsPerBlock / 16},
		State: dragonStateIdle,
		JumpProfile: newAnimatedInt([]animatedIntFrame{
			{Value: bpSize(0, 6), Count: 5},
			{Value: bpSize(0, 4), Count: 10},
			{Value: bpSize(0, 2), Count: 10},
		}),
	}
	d.Animator = newAnimator(g.assets.animation(dragonAnimationName(dragonIdle, color)))
	if invincible {
		d.InvFrames = int(2.5 * targetFPS)
	}
	g.dragons = append(g.dragons, d)
}

func (g *game) createEnemy(x, y int, kind enemyType, dir int, withAppearance bool) *enemy {
	e := enemy{
		ID:   g.nextEntityID(),
		Pos:  position{X: x, Y: y, Dir: dir},
		Type: kind,
		Mode: enemyModeNormal,
	}
	switch kind {
	case enemyCan, enemyGhost, enemyMushroom, enemyPotato, enemySnowman, enemyWitch:
		anim := newAnimator(g.assets.animation(enemyAnimationName(kind, enemyNormal)))
		e.Walking = &walkingEnemy{WalkingDir: dir, Animator: anim}
		e.Actor = walkingActor{FallSpeed: unitsPerBlock / 16, JumpSpeed: 3 * unitsPerBlock / 16}
	case enemyPurpleGhost, enemyPig:
		flyingDir := flyingDownRight
		if dir < 0 {
			flyingDir = flyingDownLeft
		}
		e.Flying = &flyingEnemy{Dir: flyingDir, Animator: newAnimator(g.assets.animation(enemyAnimationName(kind, enemyNormal)))}
	case enemyBoss:
		e.Boss = &bossEnemy{
			Animator: newAnimator(g.assets.animation(enemyAnimationName(enemyBoss, enemyNormal))),
			XDir:     -1,
			YDir:     -1,
			PopDelay: 1,
			State:    enemyNormal,
		}
	}
	if withAppearance {
		targetOffset := bpSize(-26, 0)
		actualOffset := targetOffset
		offsetCutOff := bpSize(-2, 0)
		if y+targetOffset < offsetCutOff {
			actualOffset = offsetCutOff - y
		}
		baseDelay := int(targetFPS * 1.5)
		delayAddition := (actualOffset - targetOffset) / 2
		e.Appearance = &enemyAppearance{
			Animator:     newAnimator(g.assets.animation(enemyAnimationName(kind, enemyNormal))),
			TargetX:      x,
			TargetY:      y,
			YOffset:      actualOffset,
			WaitingDelay: baseDelay + delayAddition,
		}
		e.Pos.Y = y + actualOffset
	}
	g.enemies = append(g.enemies, e)
	return &g.enemies[len(g.enemies)-1]
}

func (g *game) createItem(pos vec2i, kind itemType) {
	g.items = append(g.items, item{ID: g.nextEntityID(), Pos: position{X: pos.X, Y: pos.Y, Dir: -1}, Type: kind})
}

func (g *game) Update() error {
	g.audio.playMusic("main-theme")
	switch g.state {
	case stateTitle:
		g.updateTitle()
	case stateGameplay:
		g.updateGameplay()
	}
	g.audio.update()
	return nil
}

func (g *game) updateTitle() {
	g.timeCounter++
	titleCounter := g.timeCounter % 30
	if len(g.titleSprites) > g.titleEntity {
		if (g.timeCounter/30)%2 == 0 {
			g.titleSprites[g.titleEntity].Color = color.RGBA{R: 255, G: uint8(255 - 2*titleCounter), B: uint8(255 - 5*titleCounter), A: 255}
		} else {
			g.titleSprites[g.titleEntity].Color = color.RGBA{R: 255, G: uint8(255 + 2*(titleCounter-30)), B: uint8(255 + 5*(titleCounter-30)), A: 255}
		}
	}
	if anyKeyPressed() {
		g.startGameplay()
	}
}

func (g *game) updateStaticGameplayKeys() {
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		g.levelNumber++
		g.startLevel()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		g.levelNumber--
		if g.levelNumber < 1 {
			g.levelNumber = 1
		}
		g.startLevel()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		g.levelNumber = 101
		g.startLevel()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		g.levelNumber = 1
		g.startLevel()
	}
	if inpututil.IsKeyJustPressed(ebiten.Key1) {
		g.playerCount = 1
		g.startLevel()
	}
	if inpututil.IsKeyJustPressed(ebiten.Key2) {
		g.playerCount = 2
		g.startLevel()
	}
}

func anyKeyPressed() bool {
	if len(inpututil.AppendJustPressedKeys(nil)) > 0 {
		return true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		return true
	}
	return len(inpututil.AppendJustPressedTouchIDs(nil)) > 0
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(colorBlack)
	switch g.state {
	case stateTitle:
		g.drawTitle(screen)
	case stateGameplay:
		g.drawGameplay(screen)
	}
}

func (g *game) Layout(_, _ int) (int, int) {
	return screenWidth, screenHeight
}

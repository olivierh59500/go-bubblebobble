package main

import "image/color"

type dragonColor int

const (
	dragonGreen dragonColor = iota
	dragonBlue
)

type dragonAnimationType int

const (
	dragonIdle dragonAnimationType = iota
	dragonWalking
	dragonShooting
	dragonJumping
	dragonFalling
	dragonHit
	dragonHitStare
	dragonRespawn
)

type dragonState int

const (
	dragonStateIdle dragonState = iota
	dragonStateWalking
	dragonStateShooting
	dragonStateJumping
	dragonStateFalling
)

type walkingActor struct {
	FallSpeed       int
	JumpSpeed       int
	IgnoreCollision bool
	JumpFrameCount  int
}

func (w walkingActor) jumping() bool {
	return w.JumpFrameCount > 0
}

type dragon struct {
	ID          int
	Pos         position
	Color       dragonColor
	Actor       walkingActor
	Animator    animator
	State       dragonState
	InvFrames   int
	ShootDelay  int
	JumpProfile animatedInt
	Hit         *dragonHitState
}

type dragonHitState struct {
	Animator        animator
	State           int
	RepetitionCount int
}

func dragonAnimationName(animation dragonAnimationType, color dragonColor) string {
	prefix := "DragonGreen"
	if color == dragonBlue {
		prefix = "DragonBlue"
	}
	switch animation {
	case dragonIdle:
		return prefix + "-Idle"
	case dragonWalking:
		return prefix + "-Walking"
	case dragonShooting:
		return prefix + "-Shooting"
	case dragonJumping:
		return prefix + "-Jumping"
	case dragonFalling:
		return prefix + "-Falling"
	case dragonHit:
		return prefix + "-Hit"
	case dragonHitStare:
		return prefix + "-HitStare"
	case dragonRespawn:
		return prefix + "-Respawn"
	default:
		return prefix + "-Idle"
	}
}

func dragonStartingPosition(color dragonColor) position {
	if color == dragonGreen {
		return position{X: bpSize(3, 0), Y: bpSize(23, 0), Dir: 1}
	}
	return position{X: bpSize(levelWidth-1, 0), Y: bpSize(23, 0), Dir: -1}
}

type enemyType int

const (
	enemyCan enemyType = iota
	enemyGhost
	enemyPurpleGhost
	enemyPig
	enemyMushroom
	enemySnowman
	enemyPotato
	enemyWitch
	enemyBoss
)

type enemyAnimationType int

const (
	enemyNormal enemyAnimationType = iota
	enemyShooting
	enemyBubbled
	enemyItem
)

type enemyMode int

const (
	enemyModeNormal enemyMode = iota
	enemyModeBubbled
	enemyModePopping
	enemyModeBossBubbled
	enemyModeBossItem
)

type walkingEnemy struct {
	WalkingDir           int
	Animator             animator
	GapJumping           bool
	FreezeXPosDuration   int
	FreezeState          int
	ShootCooldown        int
	JumpTurnAroundsCount int
}

func (w walkingEnemy) freezing() bool {
	return w.FreezeXPosDuration > 0
}

type flyingDirection int

const (
	flyingUpRight flyingDirection = iota
	flyingDownRight
	flyingDownLeft
	flyingUpLeft
)

type flyingEnemy struct {
	Dir      flyingDirection
	Animator animator
}

func (f flyingEnemy) xDir() int {
	switch f.Dir {
	case flyingUpRight, flyingDownRight:
		return 1
	default:
		return -1
	}
}

func (f flyingEnemy) yDir() int {
	switch f.Dir {
	case flyingDownRight, flyingDownLeft:
		return 1
	default:
		return -1
	}
}

func (f *flyingEnemy) flipX() {
	switch f.Dir {
	case flyingUpRight:
		f.Dir = flyingUpLeft
	case flyingDownRight:
		f.Dir = flyingDownLeft
	case flyingDownLeft:
		f.Dir = flyingDownRight
	case flyingUpLeft:
		f.Dir = flyingUpRight
	}
}

func (f *flyingEnemy) flipY() {
	switch f.Dir {
	case flyingUpRight:
		f.Dir = flyingDownRight
	case flyingDownRight:
		f.Dir = flyingUpRight
	case flyingDownLeft:
		f.Dir = flyingUpLeft
	case flyingUpLeft:
		f.Dir = flyingDownLeft
	}
}

func (f flyingEnemy) verticalCollider() collider {
	switch f.Dir {
	case flyingUpRight:
		return flyingTopRightVertical
	case flyingDownRight:
		return flyingBottomRightVertical
	case flyingDownLeft:
		return flyingBottomLeftVertical
	default:
		return flyingTopLeftVertical
	}
}

func (f flyingEnemy) horizontalCollider() collider {
	switch f.Dir {
	case flyingUpRight:
		return flyingTopRightHorizontal
	case flyingDownRight:
		return flyingBottomRightHorizontal
	case flyingDownLeft:
		return flyingBottomLeftHorizontal
	default:
		return flyingTopLeftHorizontal
	}
}

type bossEnemy struct {
	Animator animator
	XDir     int
	YDir     int
	PopDelay int
	State    enemyAnimationType
}

type enemyAppearance struct {
	Animator     animator
	TargetX      int
	TargetY      int
	YOffset      int
	WaitingDelay int
}

type enemy struct {
	ID         int
	Pos        position
	Type       enemyType
	Mode       enemyMode
	Actor      walkingActor
	Walking    *walkingEnemy
	Flying     *flyingEnemy
	Boss       *bossEnemy
	Appearance *enemyAppearance
	Float      bubbleFloat
	Pop        bubblePop
	PoppedTag  bool
}

func enemyTypeFromTile(tile levelTileType) enemyType {
	switch tile {
	case tileEnemyCanLeft, tileEnemyCanRight:
		return enemyCan
	case tileEnemyGhostLeft, tileEnemyGhostRight:
		return enemyGhost
	case tileEnemyPurpleLeft, tileEnemyPurpleRight:
		return enemyPurpleGhost
	case tileEnemyPigLeft, tileEnemyPigRight:
		return enemyPig
	case tileEnemyMushroomLeft, tileEnemyMushroomRight:
		return enemyMushroom
	case tileEnemySnowmanLeft, tileEnemySnowmanRight:
		return enemySnowman
	case tileEnemyPotatoLeft, tileEnemyPotatoRight:
		return enemyPotato
	case tileEnemyWitchLeft, tileEnemyWitchRight:
		return enemyWitch
	default:
		return enemyCan
	}
}

func directionFromEnemyTile(tile levelTileType) int {
	switch tile {
	case tileEnemyCanRight, tileEnemyGhostRight, tileEnemyPurpleRight, tileEnemyPigRight, tileEnemyMushroomRight, tileEnemySnowmanRight, tileEnemyPotatoRight, tileEnemyWitchRight:
		return 1
	default:
		return -1
	}
}

func enemyAnimationName(kind enemyType, animation enemyAnimationType) string {
	switch animation {
	case enemyNormal:
		switch kind {
		case enemyCan:
			return "Can-Walk"
		case enemyGhost:
			return "Ghost-Walk"
		case enemyPurpleGhost:
			return "Purple-Fly"
		case enemyPig:
			return "Pig-Fly"
		case enemyMushroom:
			return "Mushroom-Jump"
		case enemyPotato:
			return "Potato-Walk"
		case enemyWitch:
			return "Witch-Walk"
		case enemySnowman:
			return "Snowman-Walk"
		case enemyBoss:
			return "Boss-Walk"
		}
	case enemyShooting:
		switch kind {
		case enemyGhost:
			return "Ghost-Shoot"
		case enemyPotato:
			return "Potato-Walk"
		case enemyWitch:
			return "Witch-Shoot"
		case enemySnowman:
			return "Snowman-Walk"
		case enemyBoss:
			return "Boss-Walk"
		}
	case enemyBubbled:
		switch kind {
		case enemyCan:
			return "Can-Bubbled"
		case enemyGhost:
			return "Ghost-Bubbled"
		case enemyPurpleGhost:
			return "Purple-Bubbled"
		case enemyPig:
			return "Pig-Bubbled"
		case enemyMushroom:
			return "Mushroom-Bubbled"
		case enemyPotato:
			return "Potato-Bubbled"
		case enemyWitch:
			return "Witch-Bubbled"
		case enemySnowman:
			return "Snowman-Bubbled"
		case enemyBoss:
			return "Boss-Bubbled"
		}
	case enemyItem:
		switch kind {
		case enemyCan:
			return "Can-Item"
		case enemyGhost:
			return "Ghost-Item"
		case enemyPurpleGhost:
			return "Purple-Item"
		case enemyPig:
			return "Pig-Item"
		case enemyMushroom:
			return "Mushroom-Item"
		case enemyPotato:
			return "Potato-Item"
		case enemyWitch:
			return "Witch-Item"
		case enemySnowman:
			return "Snowman-Item"
		case enemyBoss:
			return "Boss-Item"
		}
	}
	return "Can-Walk"
}

func enemyProjectileAnimationName(kind enemyType) string {
	switch kind {
	case enemyGhost:
		return "Projectile-Ghost"
	case enemyPotato:
		return "Projectile-Potato"
	case enemyWitch:
		return "Projectile-Witch"
	case enemySnowman:
		return "Projectile-Snowman"
	default:
		return "Projectile-Ghost"
	}
}

type itemType int

const (
	itemBanana itemType = iota
	itemApple
	itemPear
	itemMelon
	itemGrapes
	itemPineapple
	itemSoup
	itemMeal
	itemShoe
	itemPotion
	itemToyFlamingo
	itemDoor
	itemBook
	itemElementCount
)

type item struct {
	ID   int
	Pos  position
	Type itemType
}

func itemSpriteName(item itemType) string {
	switch item {
	case itemBanana:
		return "Item-Banana"
	case itemApple:
		return "Item-Apple"
	case itemPear:
		return "Item-Pear"
	case itemMelon:
		return "Item-Melon"
	case itemGrapes:
		return "Item-Grapes"
	case itemPineapple:
		return "Item-Pineapple"
	case itemShoe:
		return "Item-Shoe"
	case itemToyFlamingo:
		return "Item-ToyFlamingo"
	case itemBook:
		return "Item-Book"
	case itemSoup:
		return "Item-Soup"
	case itemMeal:
		return "Item-Meal"
	case itemPotion:
		return "Item-Poison-Green"
	case itemDoor:
		return "Item-Door"
	default:
		return "Item-Banana"
	}
}

func itemPoints(item itemType) int {
	switch item {
	case itemBanana:
		return 500
	case itemApple:
		return 1000
	case itemPear:
		return 2000
	case itemMelon:
		return 3000
	case itemGrapes:
		return 6000
	case itemPineapple:
		return 8000
	case itemShoe, itemPotion:
		return 1000
	case itemSoup, itemMeal:
		return 2000
	default:
		return 0
	}
}

func itemOfLevel(level int) itemType {
	if level >= int(itemElementCount) {
		return itemPineapple
	}
	return itemType(level)
}

func pointTextSpriteName(item itemType) string {
	switch item {
	case itemBanana:
		return "Points-500"
	case itemApple, itemShoe, itemPotion:
		return "Points-1000"
	case itemPear, itemSoup, itemMeal:
		return "Points-2000"
	case itemMelon:
		return "Points-3000"
	case itemGrapes:
		return "Points-6000"
	case itemPineapple:
		return "Points-8000"
	default:
		return "Points-500"
	}
}

func itemTypeFromTile(tile levelTileType) itemType {
	switch tile {
	case tileItemSoup:
		return itemSoup
	case tileItemMeal:
		return itemMeal
	case tileItemShoe:
		return itemShoe
	case tileItemPotion:
		return itemPotion
	case tileItemDoor:
		return itemDoor
	case tileItemFlamingo:
		return itemToyFlamingo
	default:
		return itemMeal
	}
}

type bubbleKind int

const (
	bubbleShooting bubbleKind = iota
	bubbleFloating
	bubblePopping
)

type bubbleLeader struct {
	GroupLeader            [2]int
	CurrentIndex           int
	TimeToSwitchGroupIndex int
	Initialized            bool
}

func (b *bubbleLeader) init(leader int) {
	b.GroupLeader[0] = leader
	b.GroupLeader[1] = leader
	b.TimeToSwitchGroupIndex = 10
	b.Initialized = true
}

func (b bubbleLeader) leader() int {
	return b.GroupLeader[b.CurrentIndex]
}

func (b *bubbleLeader) setLeader(leader int) {
	b.GroupLeader[b.CurrentIndex] = leader
}

func (b *bubbleLeader) updateAndSwitch() bool {
	b.TimeToSwitchGroupIndex--
	if b.TimeToSwitchGroupIndex == 0 {
		b.TimeToSwitchGroupIndex = 10
		b.CurrentIndex = 1 - b.CurrentIndex
		return true
	}
	return false
}

func (b bubbleLeader) sharesLeader(other bubbleLeader) bool {
	return min(b.GroupLeader[0], b.GroupLeader[1]) == min(other.GroupLeader[0], other.GroupLeader[1])
}

type bubbleShoot struct {
	Direction          int
	Animator           animator
	ShootFrame         int
	PopableDelayFrame  int
	JumpableDelayFrame int
	State              int
	IgnoreCollision    bool
	IgnoreWaitFrame    int
}

func (b bubbleShoot) waiting() bool {
	return b.JumpableDelayFrame > 0
}

type bubbleFloat struct {
	Animator      animator
	LifeFrame     int
	Leader        bubbleLeader
	PopFrame      int
	WaitingForPop bool
}

type bubblePop struct {
	Animator           animator
	PrePop             bool
	Repetitions        int
	PoppedFromLifetime bool
	GavePoints         bool
	ItemLevel          int
}

type bubble struct {
	ID    int
	Pos   position
	Kind  bubbleKind
	Shoot bubbleShoot
	Float bubbleFloat
	Pop   bubblePop
}

type projectileState int

const (
	projectileShooting projectileState = iota
	projectileReversing
	projectileDestroyed
)

type projectile struct {
	ID              int
	Pos             position
	ShooterType     enemyType
	Animator        animator
	State           projectileState
	DistanceMoved   int
	StartX          int
	DestroyedTagged bool
}

type tumble struct {
	ID              int
	Pos             position
	Animator        animator
	XVel            animatedInt
	YVel            animatedInt
	ItemToSpawn     itemType
	IgnoreCollision bool
	Falling         bool
}

type positionAnimation struct {
	Start      vec2i
	End        vec2i
	TotalFrame int
	Progress   int
}

type floatingText struct {
	ID        int
	Pos       position
	Sprite    spriteHandle
	Animation positionAnimation
}

type simpleSprite struct {
	Pos    position
	Sprite spriteHandle
	Color  color.RGBA
	ScaleX float64
	ScaleY float64
}

type uiText struct {
	Pos      vec2i
	Text     string
	Color    color.RGBA
	FontSize int
	Spacing  int
}

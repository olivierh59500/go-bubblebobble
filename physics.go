package main

func xPosToTileCoord(x int) int {
	return x/unitsPerBlock - 2
}

func yPosToTileCoord(y int) int {
	return y / unitsPerBlock
}

func (g *game) collisionAt(blockX, blockY int) bool {
	if blockX < 0 || blockX >= levelWidth {
		return true
	}
	if blockY < 0 || blockY >= levelHeight || g.level == nil {
		return false
	}
	return !g.level.Tiles.isEmpty(blockX, blockY)
}

func (g *game) collidesWithWall(pos position, col collider) bool {
	if g.level == nil {
		return false
	}
	v := pos.vec()
	for x := xPosToTileCoord(col.left(v)); x <= xPosToTileCoord(col.right(v)-1); x++ {
		for y := yPosToTileCoord(col.top(v)); y <= yPosToTileCoord(col.bottom(v)-1); y++ {
			if x < 0 || x >= levelWidth || (y >= 0 && y < levelHeight && !g.level.Tiles.isEmpty(x, y)) {
				return true
			}
		}
	}
	return false
}

func (g *game) getAirflowDirection(col collider, pos vec2i) vec2i {
	if g.level == nil {
		return vec2i{}
	}
	dir := vec2i{}
	for x := xPosToTileCoord(col.left(pos)); x <= xPosToTileCoord(col.right(pos)); x++ {
		for y := yPosToTileCoord(col.top(pos)); y <= yPosToTileCoord(col.bottom(pos)); y++ {
			dir = dir.add(airflowToDirection(g.level.Airflow.get(x, y, true)))
		}
	}
	dir.normalizeSign()
	return dir
}

func airflowToDirection(tile levelTileType) (int, int) {
	switch tile {
	case tileAirflowUp:
		return 0, -1
	case tileAirflowDown:
		return 0, 1
	case tileAirflowRight:
		return 1, 0
	case tileAirflowLeft:
		return -1, 0
	default:
		return 0, 0
	}
}

func calculateMovementToRoundedPosition(pos position, col collider, dir int) int {
	coord := pos.X + col.OffsetX
	if dir > 0 {
		coord += col.W
	}
	if coord%unitsPerBlock == 0 {
		return 0
	}
	if dir > 0 {
		return unitsPerBlock - coord%unitsPerBlock
	}
	return -(coord % unitsPerBlock)
}

func shouldWalkingActorIgnoreCollisions(g *game, pos position, col collider) bool {
	if pos.Y > 24*unitsPerBlock || pos.Y <= -2*unitsPerBlock {
		return true
	}
	return g.collidesWithWall(pos, col)
}

func isWalkingActorGrounded(g *game, pos position, actor walkingActor) bool {
	pos.Y += actor.FallSpeed
	return !actor.IgnoreCollision && g.collidesWithWall(pos, walkingActorCollider)
}

func shouldWalkingEnemyGapJump(g *game, pos position, dir int) bool {
	if dir == 0 {
		return false
	}
	var blockX int
	if dir == -1 {
		blockX = xPosToTileCoord(pos.X)
		if pos.X%unitsPerBlock == 0 {
			blockX--
		}
	} else {
		blockX = xPosToTileCoord(pos.X + bpSize(2, 0))
	}
	blockY := yPosToTileCoord(pos.Y + bpSize(2, 0))
	return !g.collisionAt(blockX, blockY) &&
		!g.collisionAt(blockX+dir, blockY) &&
		!g.collisionAt(blockX, blockY-1) &&
		!g.collisionAt(blockX+dir, blockY-1) &&
		!g.collisionAt(blockX+2*dir, blockY-1)
}

func (g *game) collidesWithDragonSpikes(pos position, col collider) bool {
	for i := range g.dragons {
		d := &g.dragons[i]
		if d.Hit != nil {
			continue
		}
		for _, spike := range dragonSpikeColliders {
			if d.Pos.Dir > 0 {
				spike = spike.flipX(bpSize(2, 0))
			}
			if overlaps(pos, col, d.Pos, spike) {
				return true
			}
		}
	}
	return false
}

func (g *game) collidesWithJumpableBubble(pos position, col collider) bool {
	for i := range g.bubbles {
		if g.bubbles[i].Kind == bubbleFloating && overlaps(pos, col, g.bubbles[i].Pos, bubbleJumpCollider) {
			return true
		}
	}
	for i := range g.enemies {
		if g.enemies[i].Mode == enemyModeBubbled && overlaps(pos, col, g.enemies[i].Pos, bubbleJumpCollider) {
			return true
		}
	}
	return false
}

func (g *game) collidingShootingBubble(pos position, col collider) *bubble {
	for i := range g.bubbles {
		b := &g.bubbles[i]
		if b.Kind == bubbleShooting && !b.Shoot.waiting() && overlaps(pos, col, b.Pos, bubbleCollider) {
			return b
		}
	}
	return nil
}

func (g *game) collidesWithEnemy(pos position, col collider) bool {
	for i := range g.enemies {
		e := &g.enemies[i]
		if e.Mode == enemyModeNormal && e.Appearance == nil && e.Boss == nil && overlaps(pos, col, e.Pos, enemyHitCollider) {
			return true
		}
	}
	return false
}

func (g *game) collidesWithEnemyProjectile(pos position, col collider) bool {
	for i := range g.projectiles {
		p := &g.projectiles[i]
		if p.State != projectileDestroyed && overlaps(pos, col, p.Pos, enemyProjectileCollider) {
			return true
		}
	}
	return false
}

func (g *game) collidesWithBoss(pos position, col collider) bool {
	for i := range g.enemies {
		e := &g.enemies[i]
		if e.Boss != nil && e.Boss.State == enemyNormal && overlaps(pos, col, e.Pos, bossCollider) {
			return true
		}
	}
	return false
}

func (g *game) dragonBubblePushDirection(pos position, col collider) int {
	dir := 0
	for i := range g.dragons {
		d := &g.dragons[i]
		if d.Hit != nil {
			continue
		}
		weak := weakDragonBubblePushCollider
		strong := strongDragonBubblePushCollider
		if d.Pos.Dir == 1 {
			weak = weak.flipX(bpSize(2, 0))
			strong = strong.flipX(bpSize(2, 0))
		}
		if overlaps(pos, col, d.Pos, strong) {
			dir += 2 * d.Pos.Dir
		} else if overlaps(pos, col, d.Pos, weak) {
			dir += d.Pos.Dir
		}
	}
	return dir
}

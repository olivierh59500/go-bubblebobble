package main

const (
	freezeNone = iota
	freezeForJump
	freezeForShoot
)

const (
	bubbleStateNone = iota
	bubbleStateNormalShoot
	bubbleStateIgnoreCollisionShoot
	bubbleStateIgnoreCollisionWait
)

func (g *game) updateGameplay() {
	g.updateStaticGameplayKeys()
	g.updateEnemyAppearance()
	g.updateWalkingEnemies()
	g.updateBosses()
	g.updateProjectiles()
	g.updateFlyingEnemies()
	g.updatePopEnemyBubbles()
	g.updateTumbles()
	g.updateDragons()
	g.updateDragonHits()
	g.updateWalkingActors()
	g.updateBubbleShoot()
	g.updateBubbleFloat()
	g.updateBubblePop()
	g.updateItemPickup()
	g.updatePositionAnimations()
	g.cleanupDestroyed()
	g.updateLevelProgression()
}

func (g *game) chance(probability float64) bool {
	return g.rng.Float64() < probability
}

func (g *game) randomDirection() int {
	if g.rng.Intn(2) == 0 {
		return -1
	}
	return 1
}

func (g *game) activeDragonPos() (vec2i, bool) {
	var candidates []vec2i
	for i := range g.dragons {
		if g.dragons[i].ID > 0 && g.dragons[i].Hit == nil {
			candidates = append(candidates, g.dragons[i].Pos.vec())
		}
	}
	if len(candidates) == 0 {
		return vec2i{X: -1000}, false
	}
	return candidates[g.rng.Intn(len(candidates))], true
}

func (g *game) updateEnemyAppearance() {
	for i := range g.enemies {
		e := &g.enemies[i]
		if e.ID <= 0 || e.Appearance == nil {
			continue
		}
		a := e.Appearance
		a.Animator.update()
		if a.Animator.finished() {
			a.Animator.reset()
		}
		if a.YOffset < 0 {
			a.YOffset += bpSize(0, 2)
		} else if a.WaitingDelay > 0 {
			a.WaitingDelay--
		} else {
			e.Pos.X = a.TargetX
			e.Pos.Y = a.TargetY
			e.Appearance = nil
			continue
		}
		e.Pos.X = a.TargetX
		e.Pos.Y = a.TargetY + a.YOffset
	}
}

func (g *game) updateWalkingEnemies() {
	dragonPos, foundDragon := g.activeDragonPos()
	if !foundDragon {
		dragonPos = vec2i{X: -1000}
	}
	for i := range g.enemies {
		e := &g.enemies[i]
		if e.ID <= 0 || e.Mode != enemyModeNormal || e.Appearance != nil || e.Walking == nil {
			continue
		}

		if b := g.collidingShootingBubble(e.Pos, fullActorCollider); b != nil {
			g.makeEnemyBubbled(e)
			g.destroyBubble(b.ID)
			continue
		}

		w := e.Walking
		isMushroom := e.Type == enemyMushroom
		isSnowman := e.Type == enemySnowman
		canShoot := e.Type == enemyGhost || e.Type == enemyPotato || e.Type == enemySnowman || e.Type == enemyWitch

		freezeForJumpDuration := 15
		freezeForShootDuration := 30
		freezeTimeToShoot := 10
		shootingCooldown := 80
		switch e.Type {
		case enemyGhost:
			freezeForShootDuration = 30
			freezeTimeToShoot = 10
			shootingCooldown = 90
		case enemyPotato:
			freezeForShootDuration = 10
			freezeTimeToShoot = 0
			shootingCooldown = 30
		case enemyWitch:
			freezeForShootDuration = 100
			freezeTimeToShoot = 40
			shootingCooldown = 240
		case enemySnowman:
			freezeForShootDuration = 10
			freezeTimeToShoot = 0
			shootingCooldown = 10
		}

		moveSpeed := bpSize(0, 2)
		if isSnowman {
			moveSpeed = bpSize(0, 4)
			if e.Pos.X%4 != 0 {
				e.Pos.X = (e.Pos.X / 4) * 4
			}
		}
		fallSpeed := 2 * unitsPerBlock / 16
		normalJumpSpeed := 3 * unitsPerBlock / 16
		gapJumpSpeed := unitsPerBlock / 16
		normalJumpFrameCount := bpSize(5, 8) / normalJumpSpeed
		gapJumpFrameCount := bpSize(2, 8) / gapJumpSpeed

		dragonAbove := dragonPos.Y < e.Pos.Y
		dragonBelow := dragonPos.Y > e.Pos.Y
		dragonSameY := dragonPos.Y == e.Pos.Y
		lookingAtDragon := sign(dragonPos.X-e.Pos.X) == w.WalkingDir

		e.Actor.FallSpeed = fallSpeed
		if w.FreezeState == freezeForJump || (w.FreezeState == freezeForShoot && w.FreezeXPosDuration < freezeTimeToShoot) {
			w.Animator.goToIndex(0)
		} else {
			w.Animator.update()
			if w.Animator.finished() {
				w.Animator.reset()
			}
		}

		isGrounded := false
		if !e.Actor.jumping() {
			isGrounded = isWalkingActorGrounded(g, e.Pos, e.Actor)
			if isGrounded && e.Pos.Y%unitsPerBlock != 0 {
				e.Pos.Y = (e.Pos.Y/unitsPerBlock + 1) * unitsPerBlock
			}
		}

		if !e.Actor.jumping() && w.GapJumping {
			w.GapJumping = false
			w.WalkingDir = 0
		}
		if isGrounded {
			w.GapJumping = false
			if w.WalkingDir == 0 {
				if dragonPos.X == e.Pos.X {
					w.WalkingDir = g.randomDirection()
				} else {
					w.WalkingDir = sign(dragonPos.X - e.Pos.X)
					if g.chance(0.33) {
						w.WalkingDir *= -1
					}
				}
			}
			e.Pos.Dir = w.WalkingDir
		} else if !w.GapJumping {
			w.WalkingDir = 0
		}

		if w.ShootCooldown > 0 {
			w.ShootCooldown--
		}
		if w.FreezeXPosDuration > 0 {
			w.FreezeXPosDuration--
			switch {
			case w.FreezeXPosDuration == freezeTimeToShoot && w.FreezeState == freezeForShoot:
				g.createProjectile(e.Pos.X, e.Pos.Y, e.Pos.Dir, e.Type)
				w.ShootCooldown = shootingCooldown
			case w.FreezeXPosDuration == 0 && w.FreezeState == freezeForShoot:
				w.Animator.set(g.assets.animation(enemyAnimationName(e.Type, enemyNormal)))
				w.FreezeState = freezeNone
			case w.FreezeXPosDuration == 0 && w.FreezeState == freezeForJump:
				w.WalkingDir *= -1
				w.JumpTurnAroundsCount--
				if w.JumpTurnAroundsCount == 0 {
					e.Actor.JumpSpeed = normalJumpSpeed
					e.Actor.JumpFrameCount = normalJumpFrameCount
					w.FreezeState = freezeNone
				} else {
					w.setFreezing(freezeForJumpDuration, freezeForJump)
				}
			}
		} else if isSnowman {
			if w.ShootCooldown == 0 && g.chance(0.01) {
				w.ShootCooldown = shootingCooldown
				g.createProjectile(e.Pos.X, e.Pos.Y, e.Pos.Dir, e.Type)
			}
		} else if canShoot && isGrounded && dragonSameY && lookingAtDragon && w.ShootCooldown == 0 && g.chance(0.05) {
			w.setFreezing(freezeForShootDuration, freezeForShoot)
			w.Animator.set(g.assets.animation(enemyAnimationName(e.Type, enemyShooting)))
		}

		shouldGapJump := false
		if isGrounded && !w.freezing() {
			if isSnowman {
				shouldGapJump = false
			} else if isMushroom {
				shouldGapJump = true
			} else {
				shouldGapJump = shouldWalkingEnemyGapJump(g, e.Pos, w.WalkingDir)
			}
		}

		velX := 0
		if !w.freezing() {
			velX = moveSpeed * w.WalkingDir
		}

		e.Actor.IgnoreCollision = shouldWalkingActorIgnoreCollisions(g, e.Pos, walkingActorCollider)
		if isGrounded && !e.Actor.jumping() && !w.freezing() && !isSnowman {
			if isMushroom {
				if (dragonAbove && g.chance(0.84)) || (dragonSameY && g.chance(0.92)) || (dragonBelow && g.chance(0.98)) {
					w.Animator.reset()
					w.GapJumping = true
					e.Actor.JumpSpeed = gapJumpSpeed
					e.Actor.JumpFrameCount = gapJumpFrameCount
				} else {
					w.GapJumping = false
					w.JumpTurnAroundsCount = 3
					w.setFreezing(freezeForJumpDuration, freezeForJump)
				}
			} else {
				if shouldGapJump && ((dragonAbove && g.chance(0.05)) || (dragonSameY && g.chance(0.08)) || (dragonBelow && g.chance(0.015))) {
					w.GapJumping = true
					e.Actor.JumpSpeed = gapJumpSpeed
					e.Actor.JumpFrameCount = gapJumpFrameCount
				} else if (dragonAbove && g.chance(0.011)) || (dragonSameY && g.chance(0.005)) || (dragonBelow && g.chance(0.005)) {
					w.GapJumping = false
					w.JumpTurnAroundsCount = 3
					w.setFreezing(freezeForJumpDuration, freezeForJump)
				}
			}
		}
		if w.GapJumping && e.Actor.jumping() && e.Actor.JumpFrameCount <= gapJumpFrameCount/2 {
			e.Actor.JumpSpeed = 0
		}

		if !w.freezing() {
			e.Pos.X += velX
			if !e.Actor.IgnoreCollision && g.collidesWithWall(e.Pos, walkingActorCollider) {
				e.Pos.X = (e.Pos.X / unitsPerBlock) * unitsPerBlock
				if w.WalkingDir == -1 {
					e.Pos.X += unitsPerBlock
				}
				if w.GapJumping {
					if isMushroom {
						w.WalkingDir *= -1
					} else {
						w.WalkingDir = 0
					}
				} else {
					w.WalkingDir *= -1
					e.Pos.X -= velX
					if g.collidesWithWall(e.Pos, walkingActorCollider) {
						w.WalkingDir *= -1
					}
					e.Pos.X += velX
				}
			}
		}
	}
}

func (w *walkingEnemy) setFreezing(duration, state int) {
	w.FreezeXPosDuration = duration
	w.FreezeState = state
}

func (g *game) updateFlyingEnemies() {
	for i := range g.enemies {
		e := &g.enemies[i]
		if e.ID <= 0 || e.Mode != enemyModeNormal || e.Appearance != nil || e.Flying == nil {
			continue
		}
		if b := g.collidingShootingBubble(e.Pos, fullActorCollider); b != nil {
			g.makeEnemyBubbled(e)
			g.destroyBubble(b.ID)
			continue
		}
		f := e.Flying
		f.Animator.update()
		if f.Animator.finished() {
			f.Animator.reset()
		}
		speedX := bpSize(0, 2)
		speedY := bpSize(0, 1)
		if e.Type == enemyPurpleGhost {
			speedY = bpSize(0, 2)
		}
		xVel := speedX * f.xDir()
		yVel := speedY * f.yDir()
		e.Pos.X += xVel
		if g.collidesWithWall(e.Pos, f.verticalCollider()) || g.collidesWithWall(e.Pos, f.horizontalCollider()) {
			e.Pos.X -= xVel
			f.flipX()
		}
		e.Pos.Y += yVel
		if g.collidesWithWall(e.Pos, f.verticalCollider()) || g.collidesWithWall(e.Pos, f.horizontalCollider()) {
			e.Pos.Y -= yVel
			f.flipY()
		}
		if e.Pos.Y >= bpSize(levelHeight, 0) {
			e.Pos.Y = bpSize(-2, 4)
		} else if e.Pos.Y < bpSize(-2, 0) {
			e.Pos.Y = bpSize(levelHeight, -4)
		}
		e.Pos.Dir = f.xDir()
	}
}

func (g *game) updateBosses() {
	for i := range g.enemies {
		e := &g.enemies[i]
		if e.ID <= 0 || e.Boss == nil || e.Mode == enemyModePopping {
			continue
		}
		boss := e.Boss
		boss.Animator.update()
		if boss.Animator.finished() {
			boss.Animator.reset()
		}
		switch boss.State {
		case enemyNormal:
			if b := g.collidingShootingBubble(e.Pos, bossCollider); b != nil {
				boss.State = enemyBubbled
				boss.Animator.set(g.assets.animation(enemyAnimationName(enemyBoss, enemyBubbled)))
				boss.PopDelay = 90
				g.destroyBubble(b.ID)
				continue
			}
			speed := bpSize(0, 4)
			e.Pos.X += speed * -boss.XDir
			if e.Pos.X < bpSize(2, 0) {
				e.Pos.X = bpSize(2, 0)
				boss.XDir *= -1
			} else if e.Pos.X+bpSize(8, 0) >= bpSize(levelWidth+2, 0) {
				e.Pos.X = bpSize(levelWidth+2, 0) - bpSize(8, 0)
				boss.XDir *= -1
			}
			e.Pos.Y += speed * boss.YDir
			if e.Pos.Y < bpSize(1, 0) {
				e.Pos.Y = bpSize(1, 0)
				boss.YDir *= -1
			} else if e.Pos.Y+bpSize(8, 0) >= bpSize(levelHeight-1, 0) {
				e.Pos.Y = bpSize(levelHeight-1, 0) - bpSize(8, 0)
				boss.YDir *= -1
			}
		case enemyBubbled:
			if boss.PopDelay > 0 {
				boss.PopDelay--
			}
			if boss.PopDelay <= 0 && g.collidesWithDragonSpikes(e.Pos, bossCollider) {
				boss.State = enemyItem
				boss.Animator.set(g.assets.animation(enemyAnimationName(enemyBoss, enemyItem)))
				if e.Pos.X+bpSize(4, 0) < bpSize(levelWidth/2+2, 0) {
					boss.XDir = 1
				} else {
					boss.XDir = -1
				}
				g.createTumble(e.Pos.X, e.Pos.Y, -boss.XDir, enemyWitch, itemBook)
			}
		case enemyItem:
			speed := bpSize(0, 4)
			e.Pos.X += speed * boss.XDir
			e.Pos.Y -= speed
			if e.Pos.Y+bpSize(8, 0) < 0 {
				e.ID = -1
			}
		}
	}
}

func (g *game) updateProjectiles() {
	for i := range g.projectiles {
		p := &g.projectiles[i]
		if p.ID <= 0 {
			continue
		}
		if p.StartX < -1000 {
			p.StartX = p.Pos.X
		}
		p.Animator.update()
		if p.Animator.finished() {
			if p.State == projectileDestroyed {
				p.ID = -1
				continue
			}
			p.Animator.reset()
		}
		maxDistance := bpSize(1000, 0)
		shootSpeed := 1
		switch p.ShooterType {
		case enemyGhost:
			shootSpeed = bpSize(0, 3)
		case enemyPotato, enemySnowman:
			shootSpeed = bpSize(0, 4)
		case enemyWitch:
			shootSpeed = bpSize(0, 5)
			maxDistance = bpSize(14, 0)
		}
		velocity := shootSpeed * p.Pos.Dir
		if p.ShooterType == enemySnowman {
			p.Pos.Y += shootSpeed
			if p.Pos.Y >= bpSize(levelHeight, 0) {
				p.ID = -1
			}
		} else if p.State == projectileReversing {
			p.Pos.X -= velocity
			if (p.Pos.Dir > 0 && p.Pos.X <= p.StartX) || (p.Pos.Dir < 0 && p.Pos.X >= p.StartX) {
				p.ID = -1
			}
		} else if p.State != projectileDestroyed {
			p.Pos.X += velocity
		}
		if p.ID <= 0 || p.ShooterType == enemySnowman {
			continue
		}
		p.DistanceMoved += shootSpeed
		if ((p.DistanceMoved > maxDistance && p.State != projectileReversing) || g.collidesWithWall(p.Pos, enemyProjectileCollider)) && p.State != projectileDestroyed {
			p.Pos.X -= velocity
			switch p.ShooterType {
			case enemyGhost:
				p.Animator.set(g.assets.animation("Projectile-Ghost-Destroy"))
				p.State = projectileDestroyed
				p.DestroyedTagged = true
			case enemyWitch:
				p.State = projectileReversing
			default:
				p.ID = -1
			}
		}
	}
}

func (g *game) updateDragons() {
	for i := range g.dragons {
		d := &g.dragons[i]
		if d.ID <= 0 || d.Hit != nil {
			continue
		}
		if d.InvFrames > 0 {
			d.InvFrames--
		}
		if d.InvFrames <= 0 && (g.collidesWithEnemy(d.Pos, fullActorCollider) || g.collidesWithEnemyProjectile(d.Pos, fullActorCollider) || g.collidesWithBoss(d.Pos, fullActorCollider)) {
			g.makeDragonHit(d)
			continue
		}
		inputDir := g.xAxis(d.Color)
		inputJump := g.keyDown(actionJump, d.Color)
		inputFire := g.keyDown(actionFire, d.Color)
		moveSpeed := unitsPerBlock / 16
		if g.modifiers.SpeedUp {
			moveSpeed *= 2
			d.Pos.X = (d.Pos.X / 2) * 2
		}
		velX := moveSpeed * inputDir
		startedShooting := false
		if inputDir != 0 {
			d.Pos.Dir = inputDir
		}
		if d.ShootDelay == 0 && inputFire {
			g.createBubbleCenteredAt(d.Pos.vec().add(bpSize(1, 0), bpSize(1, 0)), d.Pos.Dir)
			d.ShootDelay = 30
			if g.modifiers.FireRate {
				d.ShootDelay /= 2
			}
			g.audio.playSound("bubble-shoot-sound")
			startedShooting = true
		} else if d.ShootDelay > 0 {
			d.ShootDelay--
		}

		d.Actor.IgnoreCollision = shouldWalkingActorIgnoreCollisions(g, d.Pos, walkingActorCollider)
		d.Pos.X += velX
		if !d.Actor.IgnoreCollision && g.collidesWithWall(d.Pos, walkingActorCollider) {
			d.Pos.X -= velX
		}
		isGrounded := false
		if !d.Actor.jumping() {
			isGrounded = isWalkingActorGrounded(g, d.Pos, d.Actor)
		}
		if inputJump && !d.Actor.jumping() && (isGrounded || g.collidesWithJumpableBubble(d.Pos, walkingActorCollider)) {
			d.JumpProfile.reset()
			d.Actor.JumpFrameCount = d.JumpProfile.Length
			g.audio.playSound("dragon-jump-sound")
		}
		if d.Actor.jumping() {
			d.Actor.JumpSpeed = d.JumpProfile.get()
			d.JumpProfile.tick()
		}

		d.Animator.update()
		if !(d.State == dragonStateShooting && !d.Animator.finished()) {
			switch {
			case startedShooting:
				d.Animator.set(g.assets.animation(dragonAnimationName(dragonShooting, d.Color)))
				d.State = dragonStateShooting
			case isGrounded && inputDir == 0 && d.State != dragonStateIdle:
				d.Animator.set(g.assets.animation(dragonAnimationName(dragonIdle, d.Color)))
				d.State = dragonStateIdle
			case isGrounded && inputDir != 0 && d.State != dragonStateWalking:
				d.Animator.set(g.assets.animation(dragonAnimationName(dragonWalking, d.Color)))
				d.State = dragonStateWalking
			case !isGrounded && d.Actor.jumping() && d.State != dragonStateJumping:
				d.Animator.set(g.assets.animation(dragonAnimationName(dragonJumping, d.Color)))
				d.State = dragonStateJumping
			case !isGrounded && !d.Actor.jumping() && d.State != dragonStateFalling:
				d.Animator.set(g.assets.animation(dragonAnimationName(dragonFalling, d.Color)))
				d.State = dragonStateFalling
			}
		}
		if d.Animator.finished() && d.State != dragonStateShooting {
			d.Animator.reset()
		}
	}
}

func (g *game) makeDragonHit(d *dragon) {
	d.Hit = &dragonHitState{
		Animator:        newAnimator(g.assets.animation(dragonAnimationName(dragonHit, d.Color))),
		State:           0,
		RepetitionCount: 4,
	}
}

func (g *game) updateDragonHits() {
	for i := range g.dragons {
		d := &g.dragons[i]
		if d.ID <= 0 || d.Hit == nil {
			continue
		}
		d.Pos.Dir = -1
		h := d.Hit
		h.Animator.update()
		if h.Animator.finished() {
			h.RepetitionCount--
			h.Animator.reset()
			if h.RepetitionCount == 0 {
				switch h.State {
				case 0:
					h.Animator.set(g.assets.animation(dragonAnimationName(dragonHitStare, d.Color)))
					h.State = 1
					h.RepetitionCount = 5
				case 1:
					h.Animator.set(g.assets.animation(dragonAnimationName(dragonRespawn, d.Color)))
					h.State = 2
					h.RepetitionCount = 1
				default:
					color := d.Color
					d.ID = -1
					g.modifiers = modifiers{}
					g.createDragon(color, true)
				}
			}
		}
	}
}

func (g *game) updateWalkingActors() {
	for i := range g.dragons {
		d := &g.dragons[i]
		if d.ID > 0 && d.Hit == nil {
			updateWalkingActor(g, &d.Pos, &d.Actor)
		}
	}
	for i := range g.enemies {
		e := &g.enemies[i]
		if e.ID > 0 && e.Mode == enemyModeNormal && e.Appearance == nil && e.Walking != nil {
			updateWalkingActor(g, &e.Pos, &e.Actor)
		}
	}
}

func updateWalkingActor(g *game, pos *position, actor *walkingActor) {
	pos.X = max(2*unitsPerBlock, pos.X)
	pos.X = min(28*unitsPerBlock, pos.X)
	velY := actor.FallSpeed
	if actor.jumping() {
		velY = -actor.JumpSpeed
	}
	pos.Y += velY
	if !actor.jumping() && !actor.IgnoreCollision && g.collidesWithWall(*pos, walkingActorCollider) {
		pos.Y = (pos.Y / unitsPerBlock) * unitsPerBlock
	}
	if actor.JumpFrameCount > 0 {
		actor.JumpFrameCount--
	}
	if pos.Y >= 27*unitsPerBlock {
		pos.Y = bpSize(-3, 0)
	} else if pos.Y < bpSize(-3, -2) {
		pos.Y = bpSize(levelHeight+1, 0)
	}
}

func (g *game) updateBubbleShoot() {
	for i := range g.bubbles {
		b := &g.bubbles[i]
		if b.ID <= 0 || b.Kind != bubbleShooting {
			continue
		}
		shootVelocity := bpSize(0, 4)
		popableDelay := 3
		jumpableDelay := 5
		b.Shoot.Animator.update()
		if b.Shoot.Animator.finished() {
			b.Shoot.Animator.set(g.assets.animation("Bubble-Green-Idle"))
		}
		if b.Shoot.State == bubbleStateNone {
			if g.collidesWithWall(b.Pos, bubbleCollider) {
				if !g.wallGapExists(b.Pos, bubbleCollider, b.Shoot.Direction) {
					b.Shoot.State = bubbleStateIgnoreCollisionWait
					b.Shoot.IgnoreCollision = true
					b.Shoot.IgnoreWaitFrame = 10
				} else {
					b.Shoot.State = bubbleStateIgnoreCollisionShoot
					b.Shoot.IgnoreCollision = true
				}
			} else {
				b.Shoot.State = bubbleStateNormalShoot
			}
		} else if b.Shoot.State == bubbleStateIgnoreCollisionShoot && b.Shoot.IgnoreCollision {
			b.Shoot.IgnoreCollision = g.collidesWithWall(b.Pos, bubbleCollider)
		}

		if b.Shoot.State == bubbleStateIgnoreCollisionWait {
			b.Shoot.IgnoreWaitFrame--
			if b.Shoot.IgnoreWaitFrame == 0 {
				g.makeBubbleFloatingAndWaiting(b)
			}
		} else if !b.Shoot.waiting() {
			dx := shootVelocity * b.Shoot.Direction
			b.Pos.X += dx
			b.Shoot.ShootFrame--
			if b.Shoot.ShootFrame == 0 {
				if g.collidesWithWall(b.Pos, bubbleCollider) {
					b.Pos.X -= dx
					b.Pos.X += calculateMovementToRoundedPosition(b.Pos, bubbleCollider, b.Shoot.Direction)
				}
				g.makeBubbleFloating(b)
			} else if !b.Shoot.IgnoreCollision && g.collidesWithWall(b.Pos, bubbleCollider) {
				b.Pos.X -= dx
				b.Pos.X += calculateMovementToRoundedPosition(b.Pos, bubbleCollider, b.Shoot.Direction)
				b.Shoot.ShootFrame = 0
				b.Shoot.JumpableDelayFrame = jumpableDelay
				b.Shoot.PopableDelayFrame = popableDelay
			}
		} else {
			if b.Shoot.PopableDelayFrame > 0 {
				b.Shoot.PopableDelayFrame--
			}
			if b.Shoot.PopableDelayFrame == 0 && g.collidesWithDragonSpikes(b.Pos, bubbleCollider) {
				g.makeBubblePop(b, false, 0)
				continue
			}
			if b.Shoot.JumpableDelayFrame > 0 {
				b.Shoot.JumpableDelayFrame--
			}
			if b.Shoot.JumpableDelayFrame == 0 {
				g.makeBubbleFloating(b)
			}
		}
	}
}

func (g *game) wallGapExists(pos position, col collider, dir int) bool {
	check := pos
	if check.X%2 != 0 {
		check.X += dir
	}
	for distance := 0; distance <= bpSize(2, 8); distance += 2 {
		check.X = pos.X + distance*dir
		if !g.collidesWithWall(check, col) {
			return true
		}
	}
	return false
}

func (g *game) updateBubbleFloat() {
	refs := g.floatableRefs()
	for _, ref := range refs {
		if ref.id() <= 0 {
			continue
		}
		f := ref.float()
		pos := ref.position()
		if f.Animator.finished() {
			f.Animator.reset()
		}
		f.Animator.update()

		airflowVelocity := g.getAirflowDirection(bubbleCollider, pos.vec())
		dragonPushVelocity := bpSize(0, 1) * g.dragonBubblePushDirection(*pos, bubbleCollider)
		repel := vec2i{}
		if !f.Leader.Initialized {
			f.Leader.init(ref.id())
		}
		if f.Leader.updateAndSwitch() {
			f.Leader.setLeader(ref.id())
		}
		for _, other := range refs {
			if ref.id() == other.id() {
				continue
			}
			of := other.float()
			if of.Leader.Initialized && overlaps(*pos, bubblePopCollider, *other.position(), bubblePopCollider) {
				f.Leader.setLeader(min(f.Leader.leader(), of.Leader.leader()))
				if overlaps(*pos, bubbleRepelCollider, *other.position(), bubbleRepelCollider) {
					repel = repel.add(pos.X-other.position().X, pos.Y-other.position().Y)
				}
			}
		}
		repel.normalizeSign()
		velocity := vec2i{
			X: airflowVelocity.X + 2*repel.X + dragonPushVelocity,
			Y: airflowVelocity.Y + 2*repel.Y,
		}
		if velocity.X == 0 && g.chance(0.05) {
			velocity.X = 2 * g.randomDirection()
		}
		if f.WaitingForPop {
			f.PopFrame--
			if f.PopFrame == 0 || g.collidesWithDragonSpikes(*pos, bubbleCollider) {
				g.popBubbleGroup(ref.id(), false)
			}
			continue
		}
		pos.X += velocity.X
		if g.collidesWithWall(*pos, bubbleCollider) {
			pos.X -= velocity.X
		}
		pos.Y += velocity.Y
		if g.collidesWithWall(*pos, bubbleCollider) {
			pos.Y -= velocity.Y
		}
		if pos.Y >= bpSize(levelHeight+1, 2) {
			pos.Y = bpSize(-2, 0)
		} else if pos.Y < bpSize(-2, -2) {
			pos.Y = bpSize(levelHeight, -2)
		}
		if f.LifeFrame > 0 {
			f.LifeFrame--
		}
		if f.LifeFrame == 0 {
			g.popBubbleGroup(ref.id(), true)
		}
		if g.collidesWithDragonSpikes(*pos, bubbleCollider) {
			g.popBubbleGroup(ref.id(), false)
		}
	}
}

func (g *game) updateBubblePop() {
	for i := range g.bubbles {
		b := &g.bubbles[i]
		if b.ID <= 0 || b.Kind != bubblePopping {
			continue
		}
		g.updatePopAnimation(&b.Pos, &b.Pop, func() { b.ID = -1 }, false)
	}
	for i := range g.enemies {
		e := &g.enemies[i]
		if e.ID <= 0 || e.Mode != enemyModePopping {
			continue
		}
		g.updatePopAnimation(&e.Pos, &e.Pop, func() { e.ID = -1 }, true)
	}
}

func (g *game) updatePopAnimation(pos *position, pop *bubblePop, destroy func(), enemy bool) {
	if !pop.GavePoints {
		pop.GavePoints = true
		if !enemy {
			g.score += 10
		}
	}
	pos.Dir = -1
	if pop.PrePop {
		pop.Animator.update()
		if pop.Animator.finished() {
			pop.PrePop = false
			pop.Animator.set(g.assets.animation("Bubble-Pop"))
		}
		return
	}
	if !pop.Animator.finished() {
		pop.Animator.update()
		return
	}
	pop.Animator.reset()
	pop.Repetitions++
	if pop.Repetitions == 2 {
		destroy()
	}
}

func (g *game) updatePopEnemyBubbles() {
	for i := range g.enemies {
		e := &g.enemies[i]
		if e.ID <= 0 || e.Mode != enemyModePopping || e.PoppedTag {
			continue
		}
		e.PoppedTag = true
		pos := e.Pos
		pos.X -= bpSize(0, 2)
		if e.Pop.PoppedFromLifetime {
			g.createEnemy(pos.X, pos.Y, e.Type, g.randomDirection(), false)
		} else {
			g.createTumble(pos.X, pos.Y, g.randomDirection(), e.Type, itemOfLevel(e.Pop.ItemLevel))
		}
	}
}

func (g *game) updateTumbles() {
	for i := range g.tumbles {
		t := &g.tumbles[i]
		if t.ID <= 0 {
			continue
		}
		if !t.Falling {
			t.Pos.X += t.XVel.get() * t.Pos.Dir
		}
		t.Pos.Y += -t.YVel.get()
		if t.Pos.X < 2*unitsPerBlock {
			t.Pos.X = 2 * unitsPerBlock
			t.Pos.Dir = 1
		} else if t.Pos.X > levelWidth*unitsPerBlock {
			t.Pos.X = levelWidth * unitsPerBlock
			t.Pos.Dir = -1
		}
		if t.Pos.Y >= 27*unitsPerBlock {
			t.Pos.Y = bpSize(-3, 0)
		} else if t.Pos.Y < bpSize(-3, -2) {
			t.Pos.Y = bpSize(levelHeight+1, 0)
		}
		if t.XVel.onLastFrame() || t.YVel.onLastFrame() {
			t.Falling = true
			if t.IgnoreCollision {
				t.IgnoreCollision = shouldWalkingActorIgnoreCollisions(g, t.Pos, fullActorCollider)
			}
		}
		if !t.Falling {
			t.XVel.tick()
			t.YVel.tick()
		} else {
			dummy := walkingActor{FallSpeed: t.YVel.get(), IgnoreCollision: t.IgnoreCollision}
			if isWalkingActorGrounded(g, t.Pos, dummy) {
				t.Pos.Y = (t.Pos.Y / unitsPerBlock) * unitsPerBlock
				g.createItem(t.Pos.vec(), t.ItemToSpawn)
				t.ID = -1
				continue
			}
		}
		t.Animator.update()
		if t.Animator.finished() {
			t.Animator.reset()
		}
	}
}

func (g *game) updateItemPickup() {
	for di := range g.dragons {
		d := &g.dragons[di]
		if d.ID <= 0 || d.Hit != nil {
			continue
		}
		for ii := range g.items {
			item := &g.items[ii]
			if item.ID <= 0 {
				continue
			}
			if !overlaps(d.Pos, fullActorCollider, item.Pos, fullActorCollider) {
				continue
			}
			item.ID = -1
			points := itemPoints(item.Type)
			if points > 0 {
				g.createItemPointsText(item.Pos.vec(), item.Type)
			}
			g.audio.playSound("item-pickup-" + itoa(g.rng.Intn(3)+1))
			g.score += points
			switch item.Type {
			case itemShoe:
				g.modifiers.SpeedUp = true
			case itemSoup:
				g.modifiers.FireRate = true
			case itemMeal:
				g.modifiers.RangeUp = true
			case itemToyFlamingo:
				g.modifiers.SpeedUp = true
				g.modifiers.FireRate = true
				g.modifiers.RangeUp = true
			case itemDoor:
				if g.levelNumber == 59 {
					g.levelNumber++
					g.startLevel()
					return
				}
			case itemBook:
				if g.levelNumber == 60 {
					g.state = stateTitle
					g.setupTitle()
					return
				}
			}
		}
	}
}

func (g *game) updatePositionAnimations() {
	for i := range g.floatTexts {
		t := &g.floatTexts[i]
		if t.ID <= 0 {
			continue
		}
		a := &t.Animation
		if a.Progress == a.TotalFrame {
			t.Pos.X = a.End.X
			t.Pos.Y = a.End.Y
			t.ID = -1
			continue
		}
		progress := float64(a.Progress) / float64(a.TotalFrame)
		t.Pos.X = a.Start.X + int(float64(a.End.X-a.Start.X)*progress)
		t.Pos.Y = a.Start.Y + int(float64(a.End.Y-a.Start.Y)*progress)
		a.Progress++
	}
}

func (g *game) updateLevelProgression() {
	if g.state != stateGameplay {
		return
	}
	if g.enemyCount() == 0 {
		if !g.waitingForNext {
			g.nextLevelCountdown = 5 * targetFPS
			g.waitingForNext = true
		}
	}
	if g.nextLevelCountdown > 0 {
		g.nextLevelCountdown--
	} else if g.waitingForNext && !g.isStoryLevel() {
		g.levelNumber++
		g.startLevel()
	}
}

func (g *game) isStoryLevel() bool {
	return g.levelNumber == 59 || g.levelNumber == 60
}

func (g *game) enemyCount() int {
	count := 0
	for i := range g.enemies {
		if g.enemies[i].ID > 0 {
			count++
		}
	}
	return count
}

func (g *game) createBubbleCenteredAt(center vec2i, dir int) {
	shootFrame := 24
	if g.modifiers.RangeUp {
		shootFrame *= 2
	}
	g.bubbles = append(g.bubbles, bubble{
		ID:   g.nextEntityID(),
		Pos:  position{X: center.X - bpSize(0, 14), Y: center.Y - bpSize(1, 0), Dir: -1},
		Kind: bubbleShooting,
		Shoot: bubbleShoot{
			Direction:  dir,
			Animator:   newAnimator(g.assets.animation("Bubble-Green-Shoot")),
			ShootFrame: shootFrame,
		},
	})
}

func (g *game) createProjectile(x, y, dir int, shooter enemyType) {
	g.projectiles = append(g.projectiles, projectile{
		ID:          g.nextEntityID(),
		Pos:         position{X: x, Y: y, Dir: dir},
		ShooterType: shooter,
		Animator:    newAnimator(g.assets.animation(enemyProjectileAnimationName(shooter))),
		StartX:      -100000,
	})
}

func (g *game) createTumble(x, y, dir int, enemyType enemyType, item itemType) {
	lowS := 20
	xLow := []animatedIntFrame{{bpSize(0, 0), 10}, {bpSize(0, 1), 10}, {bpSize(0, 2), lowS}, {bpSize(0, 3), 15}, {bpSize(0, 2), lowS}, {bpSize(0, 1), 10}, {bpSize(0, 0), 10}}
	yLow := []animatedIntFrame{{bpSize(0, 3), lowS}, {bpSize(0, 2), lowS}, {bpSize(0, 1), 10}, {bpSize(0, 0), 5}, {-bpSize(0, 1), 10}, {-bpSize(0, 2), lowS}, {-bpSize(0, 3), lowS}}
	highS := 30
	xHigh := []animatedIntFrame{{bpSize(0, 0), 15}, {bpSize(0, 1), 15}, {bpSize(0, 2), highS}, {bpSize(0, 3), 30}, {bpSize(0, 2), highS}, {bpSize(0, 1), 15}, {bpSize(0, 0), 15}}
	yHigh := []animatedIntFrame{{bpSize(0, 3), highS}, {bpSize(0, 2), highS}, {bpSize(0, 1), 15}, {bpSize(0, 0), 15}, {-bpSize(0, 1), 10}, {-bpSize(0, 2), highS}, {-bpSize(0, 3), highS}}
	xDef, yDef := xLow, yLow
	if g.rng.Intn(2) == 0 {
		xDef, yDef = xHigh, yHigh
	}
	g.tumbles = append(g.tumbles, tumble{
		ID:              g.nextEntityID(),
		Pos:             position{X: x, Y: y, Dir: dir},
		Animator:        newAnimator(g.assets.animation(enemyAnimationName(enemyType, enemyItem))),
		XVel:            newAnimatedInt(xDef),
		YVel:            newAnimatedInt(yDef),
		ItemToSpawn:     item,
		IgnoreCollision: true,
	})
}

func (g *game) createItemPointsText(pos vec2i, item itemType) {
	pos.Y += bpSize(0, 8)
	g.floatTexts = append(g.floatTexts, floatingText{
		ID:     g.nextEntityID(),
		Pos:    position{X: pos.X, Y: pos.Y, Dir: -1},
		Sprite: g.assets.mustSprite(pointTextSpriteName(item)),
		Animation: positionAnimation{
			Start:      pos,
			End:        pos.add(0, bpSize(-5, 0)),
			TotalFrame: 120,
		},
	})
}

func (g *game) makeEnemyBubbled(e *enemy) {
	e.Mode = enemyModeBubbled
	e.Float = newBubbleFloat(g.assets.animation(enemyAnimationName(e.Type, enemyBubbled)))
}

func newBubbleFloat(anim *animation) bubbleFloat {
	return bubbleFloat{
		Animator:  newAnimator(anim),
		LifeFrame: 20 * targetFPS,
		Leader: bubbleLeader{
			TimeToSwitchGroupIndex: 10,
		},
	}
}

func (g *game) makeBubbleFloating(b *bubble) {
	b.Kind = bubbleFloating
	b.Float = newBubbleFloat(g.assets.animation("Bubble-Green-Idle"))
}

func (g *game) makeBubbleFloatingAndWaiting(b *bubble) {
	g.makeBubbleFloating(b)
	b.Float.PopFrame = 35
	b.Float.WaitingForPop = true
}

func (g *game) makeBubblePop(b *bubble, fromLifetime bool, itemLevel int) {
	b.Kind = bubblePopping
	b.Pop = bubblePop{
		Animator:           newAnimator(g.assets.animation("Bubble-PrePop")),
		PrePop:             true,
		PoppedFromLifetime: fromLifetime,
		ItemLevel:          itemLevel,
	}
}

func (g *game) makeEnemyPop(e *enemy, fromLifetime bool, itemLevel int) {
	e.Mode = enemyModePopping
	e.PoppedTag = false
	e.Pop = bubblePop{
		Animator:           newAnimator(g.assets.animation("Bubble-PrePop")),
		PrePop:             true,
		PoppedFromLifetime: fromLifetime,
		ItemLevel:          itemLevel,
	}
}

func (g *game) destroyBubble(id int) {
	for i := range g.bubbles {
		if g.bubbles[i].ID == id {
			g.bubbles[i].ID = -1
			return
		}
	}
}

type floatableRef struct {
	bubble *bubble
	enemy  *enemy
}

func (f floatableRef) id() int {
	if f.bubble != nil {
		return f.bubble.ID
	}
	return f.enemy.ID
}

func (f floatableRef) position() *position {
	if f.bubble != nil {
		return &f.bubble.Pos
	}
	return &f.enemy.Pos
}

func (f floatableRef) float() *bubbleFloat {
	if f.bubble != nil {
		return &f.bubble.Float
	}
	return &f.enemy.Float
}

func (f floatableRef) isEnemy() bool {
	return f.enemy != nil
}

func (g *game) floatableRefs() []floatableRef {
	var refs []floatableRef
	for i := range g.bubbles {
		if g.bubbles[i].ID > 0 && g.bubbles[i].Kind == bubbleFloating {
			refs = append(refs, floatableRef{bubble: &g.bubbles[i]})
		}
	}
	for i := range g.enemies {
		if g.enemies[i].ID > 0 && g.enemies[i].Mode == enemyModeBubbled {
			refs = append(refs, floatableRef{enemy: &g.enemies[i]})
		}
	}
	return refs
}

func (g *game) popBubbleGroup(originID int, fromLifetime bool) {
	refs := g.floatableRefs()
	var origin *floatableRef
	for i := range refs {
		if refs[i].id() == originID {
			origin = &refs[i]
			break
		}
	}
	if origin == nil {
		return
	}
	originLeader := origin.float().Leader
	itemLevel := 0
	for _, ref := range refs {
		if ref.id() != originID && !ref.float().Leader.sharesLeader(originLeader) {
			continue
		}
		if fromLifetime && ref.isEnemy() && ref.id() != originID {
			continue
		}
		if ref.bubble != nil {
			g.makeBubblePop(ref.bubble, fromLifetime, 0)
		} else {
			level := 0
			if !fromLifetime {
				level = itemLevel
				itemLevel++
			}
			g.makeEnemyPop(ref.enemy, fromLifetime, level)
		}
	}
	if !fromLifetime && itemLevel > 0 {
		soundLevel := 1
		if itemLevel > 4 {
			soundLevel = 3
		} else if itemLevel > 1 {
			soundLevel = 2
		}
		g.audio.playSound("enemy-bubble-pop-level-" + itoa(soundLevel))
	}
}

func (g *game) cleanupDestroyed() {
	g.dragons = filterSlice(g.dragons, func(d dragon) bool { return d.ID > 0 })
	g.enemies = filterSlice(g.enemies, func(e enemy) bool { return e.ID > 0 })
	g.bubbles = filterSlice(g.bubbles, func(b bubble) bool { return b.ID > 0 })
	g.projectiles = filterSlice(g.projectiles, func(p projectile) bool { return p.ID > 0 })
	g.tumbles = filterSlice(g.tumbles, func(t tumble) bool { return t.ID > 0 })
	g.items = filterSlice(g.items, func(i item) bool { return i.ID > 0 })
	g.floatTexts = filterSlice(g.floatTexts, func(t floatingText) bool { return t.ID > 0 })
}

func filterSlice[T any](in []T, keep func(T) bool) []T {
	out := in[:0]
	for _, value := range in {
		if keep(value) {
			out = append(out, value)
		}
	}
	return out
}

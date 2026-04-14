package main

type animation struct {
	Frames          []spriteHandle
	Name            string
	FramesPerSprite int
}

type animator struct {
	Anim          *animation
	CurrentSprite int
	Frame         int
}

func newAnimator(anim *animation) animator {
	return animator{Anim: anim}
}

func (a *animator) update() {
	if a.Anim == nil || len(a.Anim.Frames) == 0 {
		return
	}
	if !a.finished() {
		a.Frame++
	}
	a.CurrentSprite = a.Frame / a.Anim.FramesPerSprite
	if a.CurrentSprite >= len(a.Anim.Frames) {
		a.CurrentSprite = len(a.Anim.Frames) - 1
	}
}

func (a animator) finished() bool {
	if a.Anim == nil {
		return true
	}
	return a.Frame >= a.Anim.FramesPerSprite*len(a.Anim.Frames)
}

func (a *animator) reset() {
	a.Frame = 0
	a.CurrentSprite = 0
}

func (a *animator) set(anim *animation) {
	a.reset()
	a.Anim = anim
}

func (a *animator) goToIndex(index int) {
	a.CurrentSprite = index
	if a.Anim != nil {
		a.Frame = index * a.Anim.FramesPerSprite
	}
}

func (a animator) sprite() spriteHandle {
	if a.Anim == nil || len(a.Anim.Frames) == 0 {
		return invalidSpriteHandle
	}
	if a.CurrentSprite < 0 || a.CurrentSprite >= len(a.Anim.Frames) {
		return a.Anim.Frames[0]
	}
	return a.Anim.Frames[a.CurrentSprite]
}

type animatedIntFrame struct {
	Value int
	Count int
}

type animatedInt struct {
	Frames       []animatedIntFrame
	Frame        int
	CurrentIndex int
	Length       int
}

func newAnimatedInt(frames []animatedIntFrame) animatedInt {
	a := animatedInt{Frames: append([]animatedIntFrame(nil), frames...)}
	for _, frame := range frames {
		a.Length += frame.Count
	}
	return a
}

func (a *animatedInt) tick() {
	if len(a.Frames) == 0 {
		return
	}
	a.Frame++
	if a.Frame == a.Frames[a.CurrentIndex].Count {
		a.Frame = 0
		a.CurrentIndex++
		if a.CurrentIndex >= len(a.Frames) {
			a.CurrentIndex = len(a.Frames) - 1
		}
	}
}

func (a animatedInt) get() int {
	if len(a.Frames) == 0 {
		return 0
	}
	return a.Frames[a.CurrentIndex].Value
}

func (a *animatedInt) reset() {
	a.Frame = 0
	a.CurrentIndex = 0
}

func (a animatedInt) onLastFrame() bool {
	return len(a.Frames) > 0 && a.CurrentIndex == len(a.Frames)-1
}

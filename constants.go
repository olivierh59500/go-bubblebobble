package main

const (
	pixelsPerBlock  = 16
	unitsPerPixel   = 1
	unitsPerBlock   = unitsPerPixel * pixelsPerBlock
	scalingFactor   = 2
	targetFPS       = 60
	levelWidth      = 28
	levelHeight     = 26
	levelTileCount  = levelWidth * levelHeight
	screenWidth     = 32 * 32
	screenHeight    = 32 * 26
	audioSampleRate = 48000
)

func bpSize(blocks, pixels int) int {
	return blocks*unitsPerBlock + pixels*unitsPerPixel
}

func levelFileNumber(level int) string {
	if level >= 100 {
		return itoa(level)
	}
	if level >= 10 {
		return "0" + itoa(level)
	}
	return "00" + itoa(level)
}

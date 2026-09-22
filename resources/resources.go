// Package resources contains the artwork, audio and campaign embedded in the game.
package resources

import "embed"

//go:embed levels.json sprites.json sprites tiles audio arcade.ttf logo.png icon.png
var Files embed.FS

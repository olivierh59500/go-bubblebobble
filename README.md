# Bubble Bobble
This is a fan remake of Bubble Bobble written in Go with Ebiten.
Note that I do not *own* any of the assets (sprites and audio) included in this repository.

## Platforms
Ebiten supports Linux, macOS, Windows and WebAssembly.

Used libraries are:
- Ebiten
- go-mp3
- golang.org/x/image

## Editor
The project uses *Tiled* levels in `res/levels`. Place tiles only on their matching layer: blocks on `Tiles`, airflow on `Airflow`, enemies on `Enemies`, and items on `Items`.

## Controls
- Single player: arrow keys to move, Space to jump, A to fire.
- Two players: press 2 during gameplay. Green uses arrows, Up to jump, L to fire. Blue uses A/D, S to jump, T to fire.
- Debug navigation: N/M move between levels, Q jumps to level 101, W returns to level 1.

## Building

Run directly:
```console
go run .
```

Build a binary:
```console
go build -o bubblebobble .
```

# Bubble Bobble

A single-player arcade game in Go, powered by Ebitengine 2.10.2. Clear 25 rounds by trapping enemies in bubbles and popping them. Choose the green or blue dragon in your profile settings.

The campaign includes all 25 tile maps and 140 enemy placements, six enemy types (Zen-Chan, Monsta, Mighta, Pulpul, Banebou and Invader), boulders and lasers, water/lightning/fire bubbles, twelve power-ups, fruit, chain bonuses, three lives, respawning, hurry-up mode, and victory and game-over screens. Menus, a pause screen, local profiles, a leaderboard, audio settings and a level editor are included.

## Run and build

Requires Go 1.26 or later. Artwork, fonts, levels, effects and music are embedded: the executable works from any directory without external resource files.

```sh
go run .
go build -o bubblebobble .
```

Ebitengine supports macOS, Windows, Linux and WebAssembly. Desktop builds use pure Go; Linux still needs runtime graphics, window-system and audio libraries.

```sh
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/bubblebobble.exe .
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/bubblebobble-linux .
make web
```

Serve `dist/web` with a local HTTP server to play the browser build; opening the HTML directly from disk will not load WebAssembly. For example: `python3 -m http.server 8080 --directory dist/web`.

## Controls

| Action | Keyboard | Standard gamepad |
| --- | --- | --- |
| Move | Left/Right or A/D | Stick or D-pad |
| Jump | Up or W | Bottom face button (A) |
| Shoot | Space or Ctrl | Left face button (X) |
| Pause/resume | Esc | Start |
| Select menu item | Up/Down or Tab | D-pad up/down |
| Confirm | Enter or Space | A |
| Back | Esc | B |
| Fullscreen | F11 | Settings menu |
| Mute | M | Settings menu |

Menus also support the mouse. Touch an established bubble to pop it; hold jump while landing on one to bounce. Adjacent bubbles pop in a chain. Esc pauses every gameplay timer. Losing window focus pauses a desktop game. Restart and End Run count an unfinished campaign as a loss.

## Bonuses and progression

Enemies become angry after 30 seconds. A trapped enemy escapes if left for too long. A cleared round leaves seven seconds to collect fruit before the next round. Clearing round 25 wins the campaign. An extra life is awarded at 30,000 points and every additional 100,000 points.

| Bonus | Earned by | Effect |
| --- | --- | --- |
| Pink candy | Blowing 35 bubbles | Longer bubble range |
| Blue candy | Popping 35 bubbles | Faster shots |
| Yellow candy | Jumping 35 times | Faster firing rate |
| Shoes | Walking 15 screen widths | Faster movement |
| Orange/red/purple parasol | Popping 15/20/25 water bubbles | Skip 3/5/7 rounds |
| Crystal ring | Collecting 3 blue candies | Points for walking |
| Amethyst ring | Collecting 3 yellow candies | Points for jumping |
| Ruby ring | Collecting 3 pink candies | Points for shooting |
| Dynamite | Popping 13 fire bubbles | Defeat every enemy |
| Clock | Popping 12 lightning bubbles | Freeze enemies for seven seconds |

Bonus counters carry across rounds. Candy, shoe and ring effects reset after a death or a new round. Pending bonuses appear at each round's bonus spawn point. Water carries enemies and the dragon along the floor, lightning travels horizontally, and fire falls and spreads across platforms.

## Profiles and saves

Up to eight profiles store name, dragon, high score, furthest round, games played/won/lost and experience. A win awards 75 XP; a loss awards 25 XP. Every 100 XP increases the profile level. Campaigns start at round 1; profiles store statistics, not an in-progress match.

Desktop saves are written atomically to `bubblebobble/save.json` under the operating system's user configuration directory. Use `go run . -data-dir /path/to/saves` to choose a different directory. The same file stores volume/fullscreen preferences and one custom level. Invalid saves are reported and left intact. Browser saves use local storage for the current site.

Start with `-mute` to avoid opening an audio device. Music and effect volumes are independently adjustable in Settings.

## Level editor

Open Level Editor from the menu. Copy any campaign round or start with an empty arena. Choose a tile, eraser, spawn marker or enemy in the toolbar, then draw with the left mouse button. The right button erases. The outer side walls are protected.

- `Ctrl+Z` / `Cmd+Z`: undo the last stroke or layout change.
- `Ctrl+S` / `Cmd+S`: save the custom level.
- `F5` or Play Test: play the current layout; Esc opens the pause menu with a return-to-editor option.
- Save/Load: persist or restore the custom level.
- Bubbles: choose water, lightning, fire or none. Tile: change the tile theme.

Place at least one enemy and leave space for the actors. Spawn markers indicate the top of an actor; the data stores the platform under its feet. A spawn obstructed by tiles is moved to the nearest free space when the level starts. Custom play tests do not modify campaign statistics.

## Validation and code layout

```sh
go test ./...
go test -race ./...
go vet ./...
go run . -mute -data-dir /tmp/bubblebobble-check \
  -smoke-scene game -smoke-round 17 -smoke-frames 180 \
  -screenshot /tmp/bubblebobble.png
```

The smoke runner opens a real window, drives a short sequence of inputs and exits. It can capture `menu`, `game`, `pause`, `win`, `gameover`, `editor`, `profiles`, `settings`, `scores`, `help` or `credits`. It does not record campaign results. Graphical checks require a display session; simulation and save tests run without a window.

- `internal/game`: deterministic simulation, collision handling, campaign loading and bonus rules; no dependency on Ebitengine.
- `internal/save`: validated profiles, preferences and custom levels, with desktop and browser storage.
- `resources`: embedded assets and editable campaign JSON.
- Root package: Ebitengine rendering, input, sound, menus and editor.

## Credits

Original game, characters, artwork and arcade audio: Taito and their respective owners. Go adaptation: MALAKH SOFTWARE, 2026. Software is distributed under the included MIT [license](LICENSE); artwork and audio ownership is separate. This is an unofficial fan remake.

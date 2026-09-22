# Bubble Bobble

An arcade game in Go, powered by Ebitengine 2.10.2, with solo play and cooperative play between two Android phones over Bluetooth. Clear 25 rounds by trapping enemies in bubbles and popping them. Choose the green or blue dragon in your solo profile settings.

The campaign includes all 25 tile maps and 140 enemy placements, six enemy types (Zen-Chan, Monsta, Mighta, Pulpul, Banebou and Invader), boulders and lasers, water/lightning/fire bubbles, twelve power-ups, fruit, chain bonuses, three lives, respawning, hurry-up mode, and victory and game-over screens. Menus, a pause screen, local profiles, a leaderboard, YM music, audio settings and a level editor are included.

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
make android
```

Serve `dist/web` with a local HTTP server to play the browser build; opening the HTML directly from disk will not load WebAssembly. For example: `python3 -m http.server 8080 --directory dist/web`.

## Android and Bluetooth

`make android` builds and verifies a debug-signed ARM64 APK at `dist/android/bubblebobble-android-arm64.apk`. No USB connection is needed and the build does not run ADB. It uses Java 17, Android SDK 36, Build Tools 36.0.0, NDK 28.2.13676358, Gradle 8.11.1 and AGP 8.10.1. Ebitengine and `ebitenmobile` are both pinned to 2.10.2. Set `ANDROID_HOME` and `JAVA_HOME` when the tools are outside their usual macOS locations.

For installation later, connect exactly one authorized phone and run `./scripts/run-android.sh`. See [Android details and device checks](docs/android.md).

On the phone, use the left pad for movement, its upper sector to jump, and the **BUBBLE** button to shoot. Two fingers can hold direction/jump and fire simultaneously. **II** pauses. The arena retains its proportions, with controls beside it in landscape and below it on narrow displays. Menus, profile creation and the level editor accept touch. Preview the controls on a computer with `go run . -touch`.

For cooperation, open **Bluetooth Co-op** on both phones. One player selects **Host a Game** and allows discoverability; the other selects **Join a Game** and chooses that phone. Android requests nearby-device permissions only when starting Bluetooth. Paired phones and nearby discoverable phones appear in the chooser. No internet connection or remote server is involved.

The host plays the green dragon and the guest plays the blue dragon. Each has three lives and independent power-ups, with a shared score and bonus counters. Both players can pop bubbles and collect items. The surviving player continues if the other runs out of lives; a team extra life revives an eliminated partner. The campaign ends when both players are eliminated. Either player can pause/resume, and the host can restart. A disconnection ends the network session and returns to the Bluetooth menu; reconnection starts a new campaign.

The host alone advances the simulation at 60 ticks per second. Guests send inputs and interpolate positions between compressed authoritative snapshots, normally 20 per second. Stale input is released after 250 ms. Bounded queues, version checks, connection deadlines and session identifiers handle slow links, cancellation and late Android callbacks. The protocol and two-player simulation are tested without radios; physical pairing, latency and suspend/resume still require two Android devices.

## YM music

Five supplied YM files are embedded and synthesized with `ym-player` revision `3f73bdca82e5`, using its `stsound` package. The synthesizer and Ebitengine audio context both run at **48,000 Hz**. A reused 4096-sample mono buffer feeds 16-bit stereo PCM without allocation in the reader. WAV effects are resampled to the same rate. Android opens the audio device from the first update after its view is ready.

The main theme uses track 1, the menus use track 2, hurry-up uses track 3, game over uses track 4, and victory uses track 5. The embedded song metadata is retained. Music and effects have independent volume controls.

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

The smoke runner opens a real window, drives a short sequence of inputs and exits. It can capture `menu`, `game`, `coop`, `pause`, `win`, `gameover`, `editor`, `profiles`, `new-profile`, `settings`, `scores`, `help`, `bluetooth` or `credits`. It does not record campaign results. Graphical checks require a display session; simulation, protocol, YM and save tests run without a window.

- `internal/game`: deterministic simulation, collision handling, campaign loading and bonus rules; no dependency on Ebitengine.
- `internal/save`: validated profiles, preferences and custom levels, with desktop and browser storage.
- `resources`: embedded assets and editable campaign JSON.
- `internal/app`: shared Ebitengine rendering, input, sound, menus, virtual controls and editor.
- `internal/ym`: reusable 48 kHz YM reader, shared by desktop, web and Android.
- `internal/netplay`: authoritative cooperative protocol over a reliable byte stream.
- `internal/platform`: synchronized Android callbacks, separate from the game thread.
- `mobile` and `android`: Ebitengine binding, native lifecycle, permissions and Bluetooth transport.
- Root package: desktop/browser entry point, including `go run .`.

## Credits

Original game, characters, artwork and arcade effects: Taito and their respective owners. The supplied YM collection credits Tim & Mike Follin, with conversion credits retained in the files; the fifth tune is the bonus-room theme with an unidentified author in its metadata. Go adaptation: MALAKH SOFTWARE, 2026. Software is distributed under the included MIT [license](LICENSE); artwork and audio ownership is separate. This is an unofficial fan remake.

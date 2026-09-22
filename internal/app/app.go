package app

import (
	"fmt"
	"image"
	"strings"
	"time"

	"bubblebobble/internal/game"
	"bubblebobble/internal/netplay"
	"bubblebobble/internal/platform"
	"bubblebobble/internal/save"
	"bubblebobble/resources"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type page int

const (
	pageTitle page = iota
	pageMenu
	pageProfiles
	pageNewProfile
	pageScores
	pageSettings
	pageHelp
	pageCredits
	pageEditor
	pagePlay
	pageResult
	pageBluetooth
)

type button struct {
	label, action string
	bounds        image.Rectangle
}

type App struct {
	renderPositions           map[int]game.Body
	renderFrame               int
	netFinalRun               int
	mobile, touchEnabled      bool
	controls                  controlsLayout
	touch                     controlsState
	touchIDs                  []ebiten.TouchID
	canvas                    *ebiten.Image
	bridge                    *platform.Bridge
	inputSerial               uint64
	ignoreTouch, platformBack bool
	peer                      *netplay.Peer
	pending                   *netplay.Pending
	netHost                   bool
	netGeneration             int64
	netRun                    int
	netToken                  []byte
	netStatus                 string
	netSounds                 []string
	playerNames               [2]string

	art             *artwork
	audio           *soundSystem
	store           *save.Store
	levels          []game.Level
	match           *game.Game
	editor          editor
	page            page
	selected, frame int
	status          string
	statusTicks     int
	name            string
	recorded        bool
	mouseX, mouseY  int
	smokeFrames     int
	capturePath     string
	captured        bool
	captureErr      error
	quit            bool
}

func New(directory string, muted bool) (*App, error) {
	levels, err := game.LoadCampaign(resources.Files)
	if err != nil {
		return nil, err
	}
	store, err := save.Open(directory)
	if err != nil {
		return nil, err
	}
	art, err := loadArtwork()
	if err != nil {
		return nil, err
	}
	audio, err := newSoundSystem(muted)
	if err != nil {
		return nil, err
	}
	audio.setVolumes(store.Data.Settings.Music, store.Data.Settings.Sound)
	a := &App{art: art, audio: audio, store: store, levels: levels, page: pageTitle, mouseX: -1, mouseY: -1, bridge: &platform.Bridge{}, canvas: ebiten.NewImage(screenWidth, screenHeight), controls: makeControls(screenWidth, screenHeight, false)}
	a.editor.level = levels[0].Clone()
	return a, nil
}

func (a *App) navigate(p page)  { a.page = p; a.selected = 0 }
func (a *App) message(s string) { a.status = s; a.statusTicks = 300 }
func (a *App) persist() {
	if err := a.store.Write(); err != nil {
		a.message("SAVE FAILED: " + err.Error())
	}
}

func (a *App) start(custom bool) {
	levels := a.levels
	if custom {
		if err := a.editor.level.Validate(); err != nil {
			a.message(err.Error())
			return
		}
		levels = []game.Level{a.editor.level.Clone()}
	}
	match, err := game.New(levels, uint64(time.Now().UnixNano()))
	if err != nil {
		a.message(err.Error())
		return
	}
	a.match = match
	a.match.Custom = custom
	a.recorded = false
	a.navigate(pagePlay)
}

func (a *App) record(won bool) {
	if a.recorded || a.match == nil || a.match.Custom || a.smokeFrames > 0 {
		return
	}
	a.recorded = true
	if err := a.store.Record(won, a.match.Score, a.match.Level.Number); err != nil {
		a.message("SAVE FAILED: " + err.Error())
	}
}

func (a *App) Update() error {
	if a.captureErr != nil {
		return a.captureErr
	}
	if a.quit || !a.mobile && ebiten.IsWindowBeingClosed() {
		a.persist()
		a.stopNetwork()
		if a.mobile {
			a.bridge.Queue("APP_EXIT")
			a.quit = false
			return nil
		}
		return ebiten.Termination
	}
	a.platformBack = a.bridge.Back.Swap(false)
	a.readTouch()
	if a.bridge.Pause.Swap(false) && a.page == pagePlay {
		a.setPaused(true)
	}
	a.updateNetwork()
	a.frame++
	if a.smokeFrames > 0 && a.frame >= a.smokeFrames {
		if a.capturePath == "" || a.captured {
			return ebiten.Termination
		}
		return nil
	}
	if a.statusTicks > 0 {
		a.statusTicks--
	}
	if pressed(ebiten.KeyF11) {
		a.store.Data.Settings.Fullscreen = !ebiten.IsFullscreen()
		ebiten.SetFullscreen(a.store.Data.Settings.Fullscreen)
		a.persist()
	}
	if pressed(ebiten.KeyM) && a.page != pageNewProfile {
		a.audio.toggleMute()
	}
	track, paused := "menu", false
	switch a.page {
	case pageTitle:
		_, _, tap := a.pointerPressed()
		if confirmPressed() || tap {
			a.navigate(pageMenu)
		}
	case pagePlay:
		if !a.mobile && a.smokeFrames == 0 && !ebiten.IsFocused() {
			a.setPaused(true)
		}
		if a.backPressed() || padButton(ebiten.StandardGamepadButtonCenterRight, true) {
			a.setPaused(!a.match.Paused)
			a.selected = 0
		} else if a.match.Paused {
			a.updateButtons()
		} else {
			in := a.controlInput()
			if a.smokeFrames > 0 {
				in = game.Input{Move: 1, Fire: true, Jump: a.frame%90 < 30}
			}
			a.advanceMatch(in)
			for _, sound := range a.match.Sounds {
				a.audio.play(sound)
				a.netSounds = append(a.netSounds, sound)
			}
			if a.match.State == game.Won || a.match.State == game.GameOver {
				a.record(a.match.State == game.Won)
				a.navigate(pageResult)
			}
		}
		track = "theme"
		if a.match.Hurry {
			track = "hurry"
		}
		paused = a.match.Paused
	case pageResult:
		track = "gameover"
		if a.match.State == game.Won {
			track = "win"
		}
		a.updateButtons()
	case pageNewProfile:
		for _, r := range ebiten.AppendInputChars(nil) {
			if len(a.name) < 16 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == ' ' || r == '-' || r == '_') {
				a.name += strings.ToUpper(string(r))
			}
		}
		n := inpututil.KeyPressDuration(ebiten.KeyBackspace)
		if len(a.name) > 0 && (n == 1 || n > 30 && n%3 == 0) {
			a.name = a.name[:len(a.name)-1]
		}
		if a.backPressed() {
			a.navigate(pageProfiles)
		} else {
			a.updateButtons()
		}
	case pageBluetooth:
		if a.backPressed() {
			a.stopNetwork()
			a.navigate(pageMenu)
		} else {
			a.updateButtons()
		}
	case pageEditor:
		if a.backPressed() {
			a.navigate(pageMenu)
		} else {
			a.updateEditor()
		}
	default:
		if a.backPressed() {
			a.navigate(pageMenu)
		} else {
			a.updateButtons()
		}
	}
	a.publishNetwork()
	return a.audio.update(track, paused)
}

func (a *App) updateButtons() {
	buttons := a.buttons()
	if len(buttons) == 0 {
		return
	}
	if pressed(ebiten.KeyArrowUp) || padButton(ebiten.StandardGamepadButtonLeftTop, true) {
		a.selected = (a.selected + len(buttons) - 1) % len(buttons)
	}
	if pressed(ebiten.KeyArrowDown, ebiten.KeyTab) || padButton(ebiten.StandardGamepadButtonLeftBottom, true) {
		a.selected = (a.selected + 1) % len(buttons)
	}
	a.selected = min(a.selected, len(buttons)-1)
	pt := a.scenePoint()
	x, y := pt.X, pt.Y
	if x != a.mouseX || y != a.mouseY {
		for i, b := range buttons {
			if image.Pt(x, y).In(b.bounds) {
				a.selected = i
			}
		}
		a.mouseX, a.mouseY = x, y
	}
	activate := confirmPressed()
	if a.page == pageNewProfile {
		activate = pressed(ebiten.KeyEnter)
	}
	if a.page == pageSettings && pressed(ebiten.KeyArrowLeft, ebiten.KeyArrowRight) {
		delta := 25
		if pressed(ebiten.KeyArrowLeft) {
			delta = -25
		}
		a.adjustSetting(buttons[a.selected].action, delta)
		return
	}
	if tx, ty, tap := a.pointerPressed(); tap {
		x, y = tx, ty
		activate = false
		for i, b := range buttons {
			if image.Pt(x, y).In(b.bounds) {
				a.selected = i
				activate = true
				break
			}
		}
	}
	if activate {
		a.activate(buttons[a.selected].action)
	}
}

func (a *App) buttons() []button {
	var b []button
	add := func(label, action string, x, y, w, h int) {
		b = append(b, button{label, action, image.Rect(x, y, x+w, y+h)})
	}
	list := func(labels, actions []string, y int) {
		for i, label := range labels {
			add(label, actions[i], 164, y+i*50, 440, 38)
		}
	}
	switch a.page {
	case pageMenu:
		add("START GAME", "start", 218, 328, 332, 44)
		labels := []string{"PROFILES", "HIGH SCORES", "LEVEL EDITOR", "SETTINGS", "HOW TO PLAY", "CREDITS"}
		actions := []string{"profiles", "scores", "editor", "settings", "help", "credits"}
		for i, label := range labels {
			add(label, actions[i], 104+(i%2)*290, 392+(i/2)*52, 270, 38)
		}
		if a.mobile {
			add("BLUETOOTH CO-OP", "bluetooth", 218, 552, 332, 38)
			add("QUIT", "quit", 264, 604, 240, 38)
		} else {
			add("QUIT", "quit", 264, 570, 240, 38)
		}
	case pageProfiles:
		for i, p := range a.store.Data.Profiles {
			label := p.Name
			if i == a.store.Data.Active {
				label = "* " + label
			}
			add(label, fmt.Sprintf("profile:%d", i), 164, 132+i*46, 440, 36)
		}
		add("NEW PROFILE", "new-profile", 164, 532, 440, 38)
		add("BACK", "menu", 264, 594, 240, 38)
	case pageNewProfile:
		for i, r := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" {
			add(string(r), "char:"+string(r), 104+(i%10)*56, 252+(i/10)*48, 48, 38)
		}
		add("SPACE", "char: ", 440, 396, 104, 38)
		add("DEL", "delete-char", 552, 396, 104, 38)
		list([]string{"CREATE PROFILE", "CANCEL"}, []string{"create-profile", "profiles"}, 520)
	case pageSettings:
		s := a.store.Data.Settings
		full := "OFF"
		if s.Fullscreen {
			full = "ON"
		}
		if a.mobile {
			list([]string{"DRAGON: " + strings.ToUpper(a.store.Profile().Avatar), fmt.Sprintf("MUSIC: %d%%", s.Music), fmt.Sprintf("SOUND: %d%%", s.Sound), "BACK"}, []string{"avatar", "music", "sound", "menu"}, 180)
		} else {
			list([]string{"DRAGON: " + strings.ToUpper(a.store.Profile().Avatar), fmt.Sprintf("MUSIC: %d%%", s.Music), fmt.Sprintf("SOUND: %d%%", s.Sound), "FULLSCREEN: " + full, "BACK"}, []string{"avatar", "music", "sound", "fullscreen", "menu"}, 180)
		}
	case pageScores, pageHelp, pageCredits:
		add("BACK", "menu", 264, 624, 240, 38)
	case pageBluetooth:
		if a.pending != nil || len(a.netToken) > 0 {
			list([]string{"CANCEL"}, []string{"bt-back"}, 370)
		} else {
			list([]string{"HOST A GAME", "JOIN A GAME", "BACK"}, []string{"bt-host", "bt-join", "bt-back"}, 320)
		}
	case pagePlay:
		if a.match.Paused {
			last := "END RUN"
			if a.match.Custom {
				last = "BACK TO EDITOR"
			}
			if a.peer != nil && !a.netHost {
				list([]string{"RESUME", last}, []string{"resume", "leave"}, 320)
			} else {
				list([]string{"RESUME", "RESTART", last}, []string{"resume", "restart", "leave"}, 320)
			}
		}
	case pageResult:
		last := "MAIN MENU"
		if a.match.Custom {
			last = "BACK TO EDITOR"
		}
		if a.peer != nil && !a.netHost {
			list([]string{last}, []string{"leave"}, 500)
		} else {
			list([]string{"PLAY AGAIN", last}, []string{"restart", "leave"}, 450)
		}
	case pageEditor:
		return a.editorButtons()
	}
	return b
}

func (a *App) activate(action string) {
	switch action {
	case "start":
		a.start(false)
	case "profiles":
		a.navigate(pageProfiles)
	case "scores":
		a.navigate(pageScores)
	case "settings":
		a.navigate(pageSettings)
	case "help":
		a.navigate(pageHelp)
	case "credits":
		a.navigate(pageCredits)
	case "editor":
		a.navigate(pageEditor)
	case "menu":
		a.navigate(pageMenu)
	case "new-profile":
		a.name = ""
		a.navigate(pageNewProfile)
	case "create-profile":
		if err := a.store.Create(a.name); err != nil {
			a.message(err.Error())
		} else {
			a.navigate(pageProfiles)
		}
	case "avatar", "music", "sound", "fullscreen":
		a.adjustSetting(action, 25)
	case "resume":
		a.setPaused(false)
	case "restart":
		if a.peer != nil {
			if a.netHost {
				a.startCoop()
			} else {
				a.message("THE HOST CAN RESTART THE MATCH")
			}
			return
		}
		a.record(false)
		a.start(a.match.Custom)
	case "leave":
		a.record(false)
		a.stopNetwork()
		if a.match.Custom {
			a.navigate(pageEditor)
		} else {
			a.navigate(pageMenu)
		}
	case "bluetooth":
		a.navigate(pageBluetooth)
	case "bt-host":
		a.beginBluetooth(true)
	case "bt-join":
		a.beginBluetooth(false)
	case "bt-back":
		a.stopNetwork()
		a.navigate(pageMenu)
	case "delete-char":
		if len(a.name) > 0 {
			a.name = a.name[:len(a.name)-1]
		}
	case "quit":
		a.quit = true
	default:
		if strings.HasPrefix(action, "char:") {
			if len(a.name) < 16 {
				a.name += strings.TrimPrefix(action, "char:")
			}
			return
		}
		if strings.HasPrefix(action, "profile:") {
			var index int
			if _, err := fmt.Sscanf(action, "profile:%d", &index); err == nil && index >= 0 && index < len(a.store.Data.Profiles) {
				a.store.Data.Active = index
				a.persist()
				a.navigate(pageMenu)
			}
		} else if a.page == pageEditor {
			a.editorAction(action)
		}
	}
}

func (a *App) adjustSetting(action string, delta int) {
	s := &a.store.Data.Settings
	switch action {
	case "avatar":
		p := a.store.Profile()
		if p.Avatar == "green" {
			p.Avatar = "blue"
		} else {
			p.Avatar = "green"
		}
	case "music":
		s.Music = (s.Music + delta + 125) % 125
	case "sound":
		s.Sound = (s.Sound + delta + 125) % 125
	case "fullscreen":
		s.Fullscreen = !s.Fullscreen
		ebiten.SetFullscreen(s.Fullscreen)
	default:
		return
	}
	a.audio.setVolumes(s.Music, s.Sound)
	a.persist()
}

func (a *App) SetupSmoke(scene string, round int) error {
	switch scene {
	case "menu":
		a.navigate(pageMenu)
	case "game", "coop", "pause", "win", "gameover":
		if round < 1 || round > len(a.levels) {
			return fmt.Errorf("invalid smoke round")
		}
		g, err := game.New(a.levels[round-1:], 1)
		if err != nil {
			return err
		}
		a.match = g
		if scene == "coop" {
			a.match, _ = game.NewPlayers(a.levels[round-1:], 1, 2)
		}
		a.navigate(pagePlay)
		if scene == "pause" {
			a.match.Paused = true
		}
		if scene == "win" || scene == "gameover" {
			a.match.State = game.GameOver
			if scene == "win" {
				a.match.State = game.Won
			}
			a.navigate(pageResult)
		}
	case "editor":
		a.navigate(pageEditor)
	case "profiles":
		a.navigate(pageProfiles)
	case "settings":
		a.navigate(pageSettings)
	case "scores":
		a.navigate(pageScores)
	case "help":
		a.navigate(pageHelp)
	case "credits":
		a.navigate(pageCredits)
	case "bluetooth":
		a.navigate(pageBluetooth)
	case "new-profile":
		a.navigate(pageNewProfile)
	default:
		return fmt.Errorf("unknown smoke scene %q", scene)
	}
	return nil
}

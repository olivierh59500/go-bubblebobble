package main

import (
	"fmt"
	"image"
	"strings"
	"time"

	"bubblebobble/internal/game"
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
)

type button struct {
	label, action string
	bounds        image.Rectangle
}

type app struct {
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

func newApp(directory string, muted bool) (*app, error) {
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
	a := &app{art: art, audio: audio, store: store, levels: levels, page: pageTitle, mouseX: -1, mouseY: -1}
	a.editor.level = levels[0].Clone()
	return a, nil
}

func (a *app) navigate(p page)  { a.page = p; a.selected = 0 }
func (a *app) message(s string) { a.status = s; a.statusTicks = 300 }
func (a *app) persist() {
	if err := a.store.Write(); err != nil {
		a.message("SAVE FAILED: " + err.Error())
	}
}

func (a *app) start(custom bool) {
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

func (a *app) record(won bool) {
	if a.recorded || a.match == nil || a.match.Custom || a.smokeFrames > 0 {
		return
	}
	a.recorded = true
	if err := a.store.Record(won, a.match.Score, a.match.Level.Number); err != nil {
		a.message("SAVE FAILED: " + err.Error())
	}
}

func (a *app) Update() error {
	if a.captureErr != nil {
		return a.captureErr
	}
	if a.quit || ebiten.IsWindowBeingClosed() {
		a.persist()
		return ebiten.Termination
	}
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
		if confirmPressed() || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			a.navigate(pageMenu)
		}
	case pagePlay:
		if a.smokeFrames == 0 && !ebiten.IsFocused() {
			a.match.Paused = true
		}
		if backPressed() || padButton(ebiten.StandardGamepadButtonCenterRight, true) {
			a.match.Paused = !a.match.Paused
			a.selected = 0
		} else if a.match.Paused {
			a.updateButtons()
		} else {
			in := playInput()
			if a.smokeFrames > 0 {
				in = game.Input{Move: 1, Fire: true, Jump: a.frame%90 < 30}
			}
			a.match.Step(in)
			for _, sound := range a.match.Sounds {
				a.audio.play(sound)
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
		track = "opening"
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
		if backPressed() {
			a.navigate(pageProfiles)
		} else {
			a.updateButtons()
		}
	case pageEditor:
		if backPressed() {
			a.navigate(pageMenu)
		} else {
			a.updateEditor()
		}
	default:
		if backPressed() {
			a.navigate(pageMenu)
		} else {
			a.updateButtons()
		}
	}
	return a.audio.update(track, paused)
}

func (a *app) updateButtons() {
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
	x, y := ebiten.CursorPosition()
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
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
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

func (a *app) buttons() []button {
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
		add("QUIT", "quit", 264, 570, 240, 38)
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
		list([]string{"CREATE PROFILE", "CANCEL"}, []string{"create-profile", "profiles"}, 388)
	case pageSettings:
		s := a.store.Data.Settings
		full := "OFF"
		if s.Fullscreen {
			full = "ON"
		}
		list([]string{"DRAGON: " + strings.ToUpper(a.store.Profile().Avatar), fmt.Sprintf("MUSIC: %d%%", s.Music), fmt.Sprintf("SOUND: %d%%", s.Sound), "FULLSCREEN: " + full, "BACK"}, []string{"avatar", "music", "sound", "fullscreen", "menu"}, 180)
	case pageScores, pageHelp, pageCredits:
		add("BACK", "menu", 264, 624, 240, 38)
	case pagePlay:
		if a.match.Paused {
			last := "END RUN"
			if a.match.Custom {
				last = "BACK TO EDITOR"
			}
			list([]string{"RESUME", "RESTART", last}, []string{"resume", "restart", "leave"}, 320)
		}
	case pageResult:
		last := "MAIN MENU"
		if a.match.Custom {
			last = "BACK TO EDITOR"
		}
		list([]string{"PLAY AGAIN", last}, []string{"restart", "leave"}, 450)
	case pageEditor:
		return a.editorButtons()
	}
	return b
}

func (a *app) activate(action string) {
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
		a.match.Paused = false
	case "restart":
		a.record(false)
		a.start(a.match.Custom)
	case "leave":
		a.record(false)
		if a.match.Custom {
			a.navigate(pageEditor)
		} else {
			a.navigate(pageMenu)
		}
	case "quit":
		a.quit = true
	default:
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

func (a *app) adjustSetting(action string, delta int) {
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

func (a *app) setupSmoke(scene string, round int) error {
	switch scene {
	case "menu":
		a.navigate(pageMenu)
	case "game", "pause", "win", "gameover":
		if round < 1 || round > len(a.levels) {
			return fmt.Errorf("invalid smoke round")
		}
		g, err := game.New(a.levels[round-1:], 1)
		if err != nil {
			return err
		}
		a.match = g
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
	default:
		return fmt.Errorf("unknown smoke scene %q", scene)
	}
	return nil
}

func (*app) Layout(int, int) (int, int) { return screenWidth, screenHeight }

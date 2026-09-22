// Package mobile binds the shared frontend to Android's Ebitengine view.
package mobile

import (
	"fmt"
	"sync"

	"bubblebobble/internal/app"
	"bubblebobble/internal/platform"
	"github.com/hajimehoshi/ebiten/v2"
	enginemobile "github.com/hajimehoshi/ebiten/v2/mobile"
)

var bridge platform.Bridge
var config struct {
	sync.Mutex
	files string
}

type bootstrap struct{ app *app.App }

func init() { ebiten.SetTPS(60); enginemobile.SetGame(&bootstrap{}) }
func (b *bootstrap) Update() error {
	if b.app == nil {
		config.Lock()
		directory := config.files
		config.Unlock()
		if directory == "" {
			return nil
		}
		a, err := app.New(directory, false)
		if err != nil {
			return fmt.Errorf("start game: %w", err)
		}
		a.EnableMobile(&bridge)
		b.app = a
	}
	return b.app.Update()
}
func (b *bootstrap) Draw(dst *ebiten.Image) {
	if b.app != nil {
		b.app.Draw(dst)
	}
}
func (b *bootstrap) Layout(w, h int) (int, int) {
	if b.app != nil {
		return b.app.Layout(w, h)
	}
	return app.ScreenWidth, app.ScreenHeight
}

func SetFilesDir(path string)              { config.Lock(); config.files = path; config.Unlock() }
func SetBluetoothAvailable(available bool) { bridge.Available.Store(available) }
func PollBluetoothCommand() string         { return bridge.Poll() }
func BluetoothStatus(generation int64, status string) {
	bridge.Post(platform.Event{Generation: generation, Kind: "status", Text: status})
}
func BluetoothClientReady(generation int64, address, name string) {
	bridge.Post(platform.Event{Generation: generation, Kind: "ready", Address: address, Text: name})
}
func BluetoothFailed(generation int64, message string) {
	bridge.Post(platform.Event{Generation: generation, Kind: "failure", Text: message})
}
func CancelBluetooth() { bridge.CancelFromPlatform() }
func Suspend()         { bridge.Suspend() }
func Back()            { bridge.Back.Store(true) }
func Dummy()           {}

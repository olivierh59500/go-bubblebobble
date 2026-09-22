//go:build js && wasm

package save

import (
	"fmt"
	"os"
	"syscall/js"
)

func dataPath(string) string { return "bubblebobble.save.v1" }
func readData(path string) (data []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("browser storage: %v", r)
		}
	}()
	v := js.Global().Get("localStorage").Call("getItem", path)
	if v.IsNull() {
		return nil, os.ErrNotExist
	}
	return []byte(v.String()), nil
}
func writeData(path string, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("browser storage: %v", r)
		}
	}()
	js.Global().Get("localStorage").Call("setItem", path, string(data))
	return nil
}

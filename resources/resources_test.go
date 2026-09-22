package resources

import (
	"bytes"
	"encoding/json"
	"image"
	_ "image/png"
	"io/fs"
	"testing"
)

func TestEmbeddedArtworkIsComplete(t *testing.T) {
	data, err := Files.ReadFile("sprites.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string][]string
	if err = json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"green/walk", "blue/walk", "zenchan/bubble", "monsta/escape", "mighta/shoot", "pulpul/angry", "banebou/dead", "invader/walk", "water-bubble", "thunder-bubble", "fire-bubble", "flame", "laser", "boulder", "crystal_ring", "food/25"} {
		if len(manifest[name]) == 0 {
			t.Fatalf("required animation %s is missing", name)
		}
	}
	for name, paths := range manifest {
		if len(paths) == 0 {
			t.Fatalf("empty animation %s", name)
		}
		for _, path := range paths {
			b, err := Files.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			im, _, err := image.Decode(bytes.NewReader(b))
			if err != nil {
				t.Fatal(err)
			}
			if im.Bounds().Empty() {
				t.Fatalf("empty frame %s", path)
			}
		}
	}
	tiles, err := fs.Glob(Files, "tiles/*.png")
	if err != nil {
		t.Fatal(err)
	}
	if len(tiles) != 25 {
		t.Fatalf("expected 25 tile themes, got %d", len(tiles))
	}
}

// Package save stores profiles, preferences and the custom level.
package save

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"bubblebobble/internal/game"
)

type Profile struct {
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
	HighScore int    `json:"high_score"`
	BestRound int    `json:"best_round"`
	Played    int    `json:"played"`
	Won       int    `json:"won"`
	Lost      int    `json:"lost"`
	XP        int    `json:"xp"`
}

type Settings struct {
	Music      int  `json:"music"`
	Sound      int  `json:"sound"`
	Fullscreen bool `json:"fullscreen"`
}

type Data struct {
	Version  int         `json:"version"`
	Profiles []Profile   `json:"profiles"`
	Active   int         `json:"active"`
	Settings Settings    `json:"settings"`
	Custom   *game.Level `json:"custom_level,omitempty"`
}

type Store struct {
	Data Data
	Path string
}

func Open(directory string) (*Store, error) {
	s := &Store{Path: dataPath(directory), Data: Data{Version: 1, Settings: Settings{Music: 50, Sound: 75}, Profiles: []Profile{{Name: "PLAYER1", Avatar: "green", BestRound: 1}}}}
	b, err := readData(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read save: %w", err)
	}
	if err = json.Unmarshal(b, &s.Data); err != nil {
		return nil, fmt.Errorf("invalid save %s: %w", s.Path, err)
	}
	if err = s.Data.Validate(); err != nil {
		return nil, fmt.Errorf("invalid save %s: %w", s.Path, err)
	}
	return s, nil
}

func (d Data) Validate() error {
	if d.Version != 1 {
		return fmt.Errorf("unsupported version %d", d.Version)
	}
	if len(d.Profiles) < 1 || len(d.Profiles) > 8 || d.Active < 0 || d.Active >= len(d.Profiles) {
		return fmt.Errorf("invalid profile selection")
	}
	seen := map[string]bool{}
	for _, p := range d.Profiles {
		if err := ValidateName(p.Name); err != nil {
			return err
		}
		name := strings.ToUpper(p.Name)
		if seen[name] {
			return fmt.Errorf("duplicate profile")
		}
		seen[name] = true
		if p.Avatar != "green" && p.Avatar != "blue" {
			return fmt.Errorf("invalid avatar")
		}
		if p.HighScore < 0 || p.BestRound < 1 || p.BestRound > 25 || p.Played < 0 || p.Won < 0 || p.Lost < 0 || p.Played != p.Won+p.Lost || p.XP < 0 {
			return fmt.Errorf("invalid profile statistics")
		}
	}
	if d.Settings.Music < 0 || d.Settings.Music > 100 || d.Settings.Sound < 0 || d.Settings.Sound > 100 {
		return fmt.Errorf("invalid audio volume")
	}
	if d.Custom != nil {
		return d.Custom.Validate()
	}
	return nil
}

func ValidateName(name string) error {
	if name != strings.TrimSpace(name) || len(name) < 1 || len(name) > 16 {
		return fmt.Errorf("use 1 to 16 characters")
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == ' ' || r == '-' || r == '_') {
			return fmt.Errorf("use letters, numbers, spaces, - or _")
		}
	}
	return nil
}

func (s *Store) Write() error {
	if err := s.Data.Validate(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.Data, "", "  ")
	if err != nil {
		return err
	}
	return writeData(s.Path, append(b, '\n'))
}

func (s *Store) Profile() *Profile { return &s.Data.Profiles[s.Data.Active] }

func (s *Store) Create(name string) error {
	name = strings.ToUpper(strings.TrimSpace(name))
	if err := ValidateName(name); err != nil {
		return err
	}
	if len(s.Data.Profiles) >= 8 {
		return fmt.Errorf("all eight profile slots are occupied")
	}
	for _, p := range s.Data.Profiles {
		if strings.EqualFold(name, p.Name) {
			return fmt.Errorf("that profile already exists")
		}
	}
	s.Data.Profiles = append(s.Data.Profiles, Profile{Name: name, Avatar: "green", BestRound: 1})
	s.Data.Active = len(s.Data.Profiles) - 1
	return s.Write()
}

func (s *Store) Record(won bool, score, round int) error {
	p := s.Profile()
	p.Played++
	if won {
		round = 25
		p.Won++
		p.XP += 75
	} else {
		p.Lost++
		p.XP += 25
	}
	p.HighScore = max(p.HighScore, score)
	p.BestRound = max(p.BestRound, min(25, max(1, round)))
	return s.Write()
}

func (s *Store) Leaderboard() []Profile {
	profiles := append([]Profile(nil), s.Data.Profiles...)
	sort.SliceStable(profiles, func(i, j int) bool {
		if profiles[i].HighScore == profiles[j].HighScore {
			return profiles[i].Won > profiles[j].Won
		}
		return profiles[i].HighScore > profiles[j].HighScore
	})
	return profiles
}

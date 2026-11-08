// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Profile struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Auth      string `json:"auth,omitempty"`
	Username  string `json:"username,omitempty"`
	AutoLogin *bool  `json:"auto_login,omitempty"`
}
type Settings struct {
	Appearance string          `json:"appearance,omitempty"`
	Profiles   []Profile       `json:"profiles"`
	Active     int             `json:"active"`
	Favorites  map[string]bool `json:"favorites"`
}

func Path() (string, error) {
	d, e := os.UserConfigDir()
	if e != nil {
		return "", e
	}
	return filepath.Join(d, "redmine-go", "settings.json"), nil
}
func Load() (Settings, error) {
	s := Settings{Favorites: map[string]bool{}}
	p, e := Path()
	if e != nil {
		return s, e
	}
	b, e := os.ReadFile(p)
	if os.IsNotExist(e) {
		return s, nil
	}
	if e != nil {
		return s, e
	}
	e = json.Unmarshal(b, &s)
	if s.Favorites == nil {
		s.Favorites = map[string]bool{}
	}
	return s, e
}
func (s Settings) Save() error {
	p, e := Path()
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		return e
	}
	b, e := json.MarshalIndent(s, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(p), "settings-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(f.Name(), p)
}
func (s Settings) FavoriteKey(server string, id int) string { return fmt.Sprintf("%s#%d", server, id) }

// CredentialID preserves old API-key entries and separates passwords by account.
func (p Profile) CredentialID() string {
	if p.Auth == "basic" {
		return p.URL + "|basic|" + p.Username
	}
	return p.URL
}

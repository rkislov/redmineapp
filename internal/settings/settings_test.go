// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProfilesNeverSerializeCredentials(t *testing.T) {
	s := Settings{Profiles: []Profile{{Name: "Work", URL: "https://example.org/redmine"}}, Favorites: map[string]bool{"https://example.org/redmine#42": true}}
	b, e := json.Marshal(s)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(b), "api_key") || strings.Contains(string(b), "password") {
		t.Fatal("credential in settings")
	}
	var restored Settings
	if e = json.Unmarshal(b, &restored); e != nil {
		t.Fatal(e)
	}
	if !restored.Favorites[s.FavoriteKey(s.Profiles[0].URL, 42)] {
		t.Fatal("favorites lost")
	}
	if s.FavoriteKey("https://another.org", 42) == s.FavoriteKey(s.Profiles[0].URL, 42) {
		t.Fatal("servers collide")
	}
}

func TestCredentialSeparationAndAppearance(t *testing.T) {
	api := Profile{URL: "https://example.org"}
	a := Profile{URL: api.URL, Auth: "basic", Username: "a"}
	b := Profile{URL: api.URL, Auth: "basic", Username: "b"}
	if api.CredentialID() != api.URL {
		t.Fatal("old API-key identity changed")
	}
	if a.CredentialID() == b.CredentialID() || a.CredentialID() == api.CredentialID() {
		t.Fatal("account credential identities overlap")
	}
	s := Settings{Appearance: "dark", Profiles: []Profile{a}}
	data, e := json.Marshal(s)
	if e != nil {
		t.Fatal(e)
	}
	var restored Settings
	if e = json.Unmarshal(data, &restored); e != nil {
		t.Fatal(e)
	}
	if restored.Appearance != "dark" || restored.Profiles[0].Auth != "basic" {
		t.Fatal("preferences lost")
	}
	if strings.Contains(string(data), "password") {
		t.Fatal("password serialized")
	}
}

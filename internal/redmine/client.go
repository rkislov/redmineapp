// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package redmine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Ref struct {
	IsClosed bool   `json:"is_closed,omitempty"`
	ID       int    `json:"id"`
	Name     string `json:"name"`
}
type User struct {
	ID        int    `json:"id"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
}
type Journal struct {
	ID      int    `json:"id"`
	User    Ref    `json:"user"`
	Notes   string `json:"notes"`
	Created string `json:"created_on"`
}
type Issue struct {
	Attachments     []Attachment `json:"attachments"`
	AllowedStatuses []Ref        `json:"allowed_statuses"`
	ID              int          `json:"id"`
	Subject         string       `json:"subject"`
	Description     string       `json:"description"`
	Project         Ref          `json:"project"`
	Tracker         Ref          `json:"tracker"`
	Status          Ref          `json:"status"`
	Priority        Ref          `json:"priority"`
	AssignedTo      Ref          `json:"assigned_to"`
	Author          Ref          `json:"author"`
	Done            int          `json:"done_ratio"`
	Start           string       `json:"start_date"`
	Due             string       `json:"due_date"`
	Updated         string       `json:"updated_on"`
	Spent           float64      `json:"spent_hours"`
	Journals        []Journal    `json:"journals"`
}
type Page struct {
	Issues []Issue `json:"issues"`
	Total  int     `json:"total_count"`
}
type Client struct {
	Base     *url.URL
	Key      string
	Username string
	Password string
	HTTP     *http.Client
}

func New(raw, key string) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(raw), "/"))
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("укажите URL Redmine, например https://redmine.example.org")
	}
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("введите API-ключ")
	}
	return &Client{Base: u, Key: strings.TrimSpace(key), HTTP: &http.Client{Timeout: 25 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

// NewBasic authenticates with the user's regular Redmine login and password.
func NewBasic(raw, username, password string) (*Client, error) {
	username = strings.TrimSpace(username)
	if username == "" || strings.Contains(username, ":") {
		return nil, fmt.Errorf("введите логин без двоеточия")
	}
	if password == "" {
		return nil, fmt.Errorf("введите пароль")
	}
	c, e := New(raw, "basic-url-validation")
	if e != nil {
		return nil, e
	}
	c.Key = ""
	c.Username = username
	c.Password = password
	return c, nil
}
func (c *Client) Do(ctx context.Context, method, path string, q url.Values, body, out any) error {
	u := *c.Base
	u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(path, "/")
	u.RawQuery = q.Encode()
	var data io.Reader
	if body != nil {
		b, e := json.Marshal(body)
		if e != nil {
			return e
		}
		data = bytes.NewReader(b)
	}
	req, e := http.NewRequestWithContext(ctx, method, u.String(), data)
	if e != nil {
		return e
	}
	if c.Username != "" {
		req.SetBasicAuth(c.Username, c.Password)
	} else {
		req.Header.Set("X-Redmine-API-Key", c.Key)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	res, e := c.HTTP.Do(req)
	if e != nil {
		return fmt.Errorf("Redmine недоступен: %w", e)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var problem struct {
			Errors []string `json:"errors"`
		}
		_ = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&problem)
		switch res.StatusCode {
		case 401:
			return fmt.Errorf("неверные данные входа или REST API выключен")
		case 403:
			return fmt.Errorf("недостаточно прав для этого действия")
		case 404:
			return fmt.Errorf("объект не найден; проверьте адрес сервера")
		}
		if len(problem.Errors) > 0 {
			return fmt.Errorf("Redmine: %s", strings.Join(problem.Errors, "; "))
		}
		return fmt.Errorf("Redmine: HTTP %d", res.StatusCode)
	}
	if out != nil && res.StatusCode != 204 {
		if e = json.NewDecoder(io.LimitReader(res.Body, 16<<20)).Decode(out); e != nil {
			return fmt.Errorf("некорректный ответ Redmine: %w", e)
		}
	}
	return nil
}
func (c *Client) Me(ctx context.Context) (User, error) {
	var v struct {
		User User `json:"user"`
	}
	e := c.Do(ctx, "GET", "users/current.json", nil, nil, &v)
	return v.User, e
}
func (c *Client) Issues(ctx context.Context, q url.Values) (Page, error) {
	var v Page
	e := c.Do(ctx, "GET", "issues.json", q, nil, &v)
	return v, e
}
func (c *Client) Issue(ctx context.Context, id int) (Issue, error) {
	var v struct {
		Issue Issue `json:"issue"`
	}
	e := c.Do(ctx, "GET", fmt.Sprintf("issues/%d.json", id), url.Values{"include": {"journals,allowed_statuses,attachments"}}, nil, &v)
	return v.Issue, e
}
func (c *Client) Update(ctx context.Context, id int, fields map[string]any) error {
	return c.Do(ctx, "PUT", fmt.Sprintf("issues/%d.json", id), nil, map[string]any{"issue": fields}, nil)
}
func (c *Client) Create(ctx context.Context, fields map[string]any) (Issue, error) {
	var v struct {
		Issue Issue `json:"issue"`
	}
	e := c.Do(ctx, "POST", "issues.json", nil, map[string]any{"issue": fields}, &v)
	return v.Issue, e
}
func (c *Client) LogTime(ctx context.Context, fields map[string]any) error {
	return c.Do(ctx, "POST", "time_entries.json", nil, map[string]any{"time_entry": fields}, nil)
}
func (c *Client) Refs(ctx context.Context, path, root string) ([]Ref, error) {
	var all []Ref
	for offset := 0; ; offset += 100 {
		var v map[string]json.RawMessage
		e := c.Do(ctx, "GET", path, url.Values{"limit": {"100"}, "offset": {fmt.Sprint(offset)}}, nil, &v)
		if e != nil {
			return nil, e
		}
		var refs []Ref
		if e = json.Unmarshal(v[root], &refs); e != nil {
			return nil, e
		}
		all = append(all, refs...)
		var total int
		_ = json.Unmarshal(v["total_count"], &total)
		if len(refs) == 0 || total == 0 || len(all) >= total {
			return all, nil
		}
	}
}

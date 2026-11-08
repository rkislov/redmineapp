// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0
package redmine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAttachmentsRoundTrip(t *testing.T) {
	for _, basic := range []bool{false, true} {
		t.Run(fmt.Sprint("basic=", basic), func(t *testing.T) {
			content := bytes.Repeat([]byte("данные\x00\xff"), 20000)
			name := "план #1 & тест.txt"
			path := filepath.Join(t.TempDir(), name)
			if e := os.WriteFile(path, content, 0600); e != nil {
				t.Fatal(e)
			}
			var attached []Upload
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if basic {
					user, pass, ok := r.BasicAuth()
					if !ok || user != "roman" || pass != "password" {
						t.Error("missing basic auth")
					}
				} else if r.Header.Get("X-Redmine-API-Key") != "secret" {
					t.Error("missing API key")
				}
				switch r.Method + " " + r.URL.Path {
				case "POST /redmine/uploads.json":
					if r.Header.Get("Content-Type") != "application/octet-stream" || r.URL.Query().Get("filename") != name {
						t.Error("invalid upload request")
					}
					b, _ := io.ReadAll(r.Body)
					if !bytes.Equal(b, content) {
						t.Error("upload corrupted")
					}
					w.WriteHeader(201)
					fmt.Fprint(w, `{"upload":{"token":"upload-token"}}`)
				case "PUT /redmine/issues/7.json":
					var v struct {
						Issue struct {
							Uploads []Upload `json:"uploads"`
							Notes   string   `json:"notes"`
						} `json:"issue"`
					}
					if e := json.NewDecoder(r.Body).Decode(&v); e != nil {
						t.Error(e)
					}
					attached = v.Issue.Uploads
					if v.Issue.Notes != "Файл" {
						t.Error("missing comment")
					}
					w.WriteHeader(204)
				case "GET /redmine/issues/7.json":
					if !strings.Contains(r.URL.Query().Get("include"), "attachments") {
						t.Error("attachments not requested")
					}
					json.NewEncoder(w).Encode(map[string]any{"issue": Issue{ID: 7, Attachments: []Attachment{{ID: 5, Filename: name, Size: int64(len(content)), ContentURL: "/redmine/attachments/download/5/file"}}}})
				case "GET /redmine/attachments/download/5/file":
					w.Write(content)
				default:
					t.Errorf("unexpected request %s", r.URL)
					w.WriteHeader(404)
				}
			}))
			defer s.Close()
			c, _ := New(s.URL+"/redmine", "secret")
			if basic {
				c, _ = NewBasic(s.URL+"/redmine", "roman", "password")
			}
			progress := int64(0)
			upload, e := c.UploadFile(context.Background(), path, func(n, total int64) {
				progress = n
				if total != int64(len(content)) {
					t.Error("wrong upload total")
				}
			})
			if e != nil {
				t.Fatal(e)
			}
			if upload.Token != "upload-token" || upload.Filename != name || progress != int64(len(content)) {
				t.Fatalf("invalid upload: %+v progress %d", upload, progress)
			}
			if e = c.Update(context.Background(), 7, map[string]any{"notes": "Файл", "uploads": []Upload{upload}}); e != nil {
				t.Fatal(e)
			}
			if len(attached) != 1 || attached[0].Token != upload.Token {
				t.Fatal("token not attached")
			}
			issue, e := c.Issue(context.Background(), 7)
			if e != nil || len(issue.Attachments) != 1 {
				t.Fatalf("attachment missing: %v", e)
			}
			var result bytes.Buffer
			if e = c.Download(context.Background(), issue.Attachments[0], &result, func(n, total int64) { progress = n }); e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(result.Bytes(), content) || progress != int64(len(content)) {
				t.Fatal("download corrupted")
			}
		})
	}
}
func TestAttachmentDownloadRejectsForeignURLsAndRedirects(t *testing.T) {
	called := false
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer foreign.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, foreign.URL+"/attachments/download/1/file", 302)
	}))
	defer server.Close()
	c, _ := New(server.URL+"/redmine", "secret")
	for _, raw := range []string{foreign.URL + "/attachments/download/1/file", server.URL + "/outside", strings.Replace(server.URL, "http:", "https:", 1) + "/redmine/attachments/1", server.URL + "/redmine/attachments/download/1/file"} {
		var dst bytes.Buffer
		if e := c.Download(context.Background(), Attachment{ContentURL: raw}, &dst, nil); e == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	if called {
		t.Fatal("foreign server received request")
	}
}
func TestAttachmentUploadErrorsAndCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(path, []byte("data"), 0600)
	for _, tc := range []struct {
		code       int
		body, want string
	}{{422, `{"errors":["Максимум 1 МБ"]}`, "Максимум 1 МБ"}, {403, "", "прав"}, {413, "", "размер"}, {201, `{"upload":{}}`, "токен"}} {
		t.Run(fmt.Sprint(tc.code, tc.want), func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.code); fmt.Fprint(w, tc.body) }))
			defer s.Close()
			c, _ := New(s.URL, "secret")
			_, e := c.UploadFile(context.Background(), path, nil)
			if e == nil || !strings.Contains(e.Error(), tc.want) {
				t.Fatalf("wrong error: %v", e)
			}
		})
	}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("cancelled request reached server") }))
	defer s.Close()
	c, _ := New(s.URL, "secret")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := c.UploadFile(ctx, path, nil); !errors.Is(e, context.Canceled) {
		t.Fatalf("not cancelled: %v", e)
	}
	if e := c.Download(ctx, Attachment{ContentURL: s.URL + "/attachments/download/1/a"}, io.Discard, nil); !errors.Is(e, context.Canceled) {
		t.Fatalf("not cancelled: %v", e)
	}
}
func TestAttachmentDownloadCancellationDuringTransfer(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("first"))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer s.Close()
	c, _ := New(s.URL, "secret")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	e := c.Download(ctx, Attachment{ContentURL: s.URL + "/attachments/download/1/file"}, io.Discard, func(n, total int64) { cancel() })
	if !errors.Is(e, context.Canceled) {
		t.Fatalf("stream not cancelled: %v", e)
	}
}

func TestDemoCreateWithAttachment(t *testing.T) {
	c := NewDemo()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "demo.txt")
	os.WriteFile(path, []byte("demo attachment"), 0600)
	upload, e := c.UploadFile(ctx, path, nil)
	if e != nil {
		t.Fatal(e)
	}
	v, e := c.Create(ctx, map[string]any{"subject": "С файлом", "uploads": []Upload{upload}})
	if e != nil || len(v.Attachments) != 1 {
		t.Fatalf("create attachments: %+v %v", v, e)
	}
	loaded, e := c.Issue(ctx, v.ID)
	if e != nil || len(loaded.Attachments) != 1 {
		t.Fatalf("load attachments: %v", e)
	}
	var data bytes.Buffer
	if e = c.Download(ctx, loaded.Attachments[0], &data, nil); e != nil || data.String() != "demo attachment" {
		t.Fatalf("download: %v %q", e, data.String())
	}
}

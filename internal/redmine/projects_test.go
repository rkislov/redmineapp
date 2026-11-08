// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package redmine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestProjectsPublishFirstPageBeforeRemainingPagesAndBoundConcurrency(t *testing.T) {
	release := make(chan struct{})
	started := make(chan int, 8)
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	var active, peak atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		if offset > 0 {
			n := active.Add(1)
			defer active.Add(-1)
			for {
				old := peak.Load()
				if n <= old || peak.CompareAndSwap(old, n) {
					break
				}
			}
			started <- offset
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
		}
		refs := []Ref{}
		for i := offset; i < min(offset+100, 501); i++ {
			refs = append(refs, Ref{ID: i + 1, Name: strconv.Itoa(i + 1)})
		}
		_ = json.NewEncoder(w).Encode(projectsPage{Projects: refs, Total: 501, Limit: 100})
	}))
	defer server.Close()
	defer unblock()
	c, _ := New(server.URL, "test")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	progress := make(chan []Ref, 8)
	done := make(chan []Ref, 1)
	errs := make(chan error, 1)
	go func() {
		refs, e := c.Projects(ctx, func(refs []Ref, total int) {
			if total != 501 {
				t.Error("wrong total")
			}
			progress <- refs
		})
		done <- refs
		errs <- e
	}()
	var first []Ref
	select {
	case first = <-progress:
	case <-ctx.Done():
		t.Fatal("first page was blocked by remaining pages")
	}
	if len(first) != 100 {
		t.Fatal("first page missing")
	}
	for i := 0; i < 4; i++ {
		select {
		case <-started:
		case <-ctx.Done():
			t.Fatal("remaining pages were not requested concurrently")
		}
	}
	unblock()
	select {
	case refs := <-done:
		if len(refs) != 501 {
			t.Fatalf("got %d projects", len(refs))
		}
		for i, v := range refs {
			if v.ID != i+1 {
				t.Fatal("out-of-order results")
			}
		}
	case <-ctx.Done():
		t.Fatal("load did not complete")
	}
	if e := <-errs; e != nil {
		t.Fatal(e)
	}
	if peak.Load() > 4 {
		t.Fatal("too many concurrent requests")
	}
	if len(first) != 100 || first[99].ID != 100 {
		t.Fatal("progress snapshot was mutated")
	}
}
func TestProjectsKeepPartialResultsOnError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("offset") != "0" {
			w.WriteHeader(500)
			return
		}
		_ = json.NewEncoder(w).Encode(projectsPage{Projects: []Ref{{ID: 1, Name: "Доступный проект"}}, Total: 2, Limit: 1})
	}))
	defer s.Close()
	c, _ := New(s.URL, "test")
	refs, e := c.Projects(context.Background(), nil)
	if e == nil || len(refs) != 1 || refs[0].ID != 1 {
		t.Fatalf("partial results lost: %v %v", refs, e)
	}
}

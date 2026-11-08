// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0

package redmine

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"sync"
)

type projectsPage struct {
	Projects []Ref `json:"projects"`
	Total    int   `json:"total_count"`
	Limit    int   `json:"limit"`
}

// Projects publishes the first page immediately, then loads at most four pages
// concurrently. Progress snapshots are independent and ordered by server offset.
func (c *Client) Projects(ctx context.Context, progress func([]Ref, int)) ([]Ref, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	fetch := func(offset int) (projectsPage, error) {
		var p projectsPage
		e := c.Do(ctx, "GET", "projects.json", url.Values{"limit": {"100"}, "offset": {fmt.Sprint(offset)}}, nil, &p)
		return p, e
	}
	first, e := fetch(0)
	if e != nil {
		return nil, e
	}
	total := max(first.Total, len(first.Projects))
	pages := map[int][]Ref{0: first.Projects}
	snapshot := func() []Ref {
		offsets := make([]int, 0, len(pages))
		for offset := range pages {
			offsets = append(offsets, offset)
		}
		sort.Ints(offsets)
		all := make([]Ref, 0)
		seen := map[int]bool{}
		for _, offset := range offsets {
			for _, v := range pages[offset] {
				if !seen[v.ID] {
					seen[v.ID] = true
					all = append(all, v)
				}
			}
		}
		return all
	}
	publish := func() []Ref {
		all := snapshot()
		if progress != nil {
			progress(all, total)
		}
		return all
	}
	all := publish()
	if len(first.Projects) == 0 || len(first.Projects) >= total {
		return all, nil
	}
	limit := first.Limit
	if limit <= 0 || limit > 100 {
		limit = len(first.Projects)
	}
	type result struct {
		offset int
		page   projectsPage
		err    error
	}
	jobs := make(chan int)
	results := make(chan result, 4)
	var workers sync.WaitGroup
	for i := 0; i < min(4, (total-1)/limit); i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for offset := range jobs {
				p, e := fetch(offset)
				select {
				case results <- result{offset, p, e}:
				case <-ctx.Done():
					return
				}
				if e != nil {
					return
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for offset := limit; offset < total; offset += limit {
			select {
			case jobs <- offset:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() { workers.Wait(); close(results) }()
	for r := range results {
		if r.err != nil {
			return all, r.err
		}
		pages[r.offset] = r.page.Projects
		all = publish()
	}
	if e := ctx.Err(); e != nil {
		return all, e
	}
	return all, nil
}

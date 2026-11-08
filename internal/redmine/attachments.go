// Copyright 2026 Кислов Роман Сергеевич
// SPDX-License-Identifier: Apache-2.0
package redmine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Attachment struct {
	ID          int    `json:"id"`
	Filename    string `json:"filename"`
	Size        int64  `json:"filesize"`
	ContentType string `json:"content_type"`
	ContentURL  string `json:"content_url"`
	Description string `json:"description"`
	Author      Ref    `json:"author"`
	Created     string `json:"created_on"`
}
type Upload struct {
	Token       string `json:"token"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type,omitempty"`
}
type TransferProgress func(done, total int64)
type progressReader struct {
	reader      io.Reader
	done, total int64
	progress    TransferProgress
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, e := r.reader.Read(p)
	r.done += int64(n)
	if r.progress != nil {
		r.progress(r.done, r.total)
	}
	return n, e
}
func responseError(res *http.Response) error {
	var v struct {
		Errors []string `json:"errors"`
	}
	_ = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&v)
	if len(v.Errors) > 0 {
		return fmt.Errorf("Redmine: %s", strings.Join(v.Errors, "; "))
	}
	switch res.StatusCode {
	case 401:
		return fmt.Errorf("неверные данные входа или REST API выключен")
	case 403:
		return fmt.Errorf("недостаточно прав для работы с вложением")
	case 404:
		return fmt.Errorf("вложение не найдено")
	case 413:
		return fmt.Errorf("файл превышает допустимый размер на сервере")
	}
	return fmt.Errorf("Redmine: HTTP %d", res.StatusCode)
}
func (c *Client) transfer(req *http.Request) (*http.Response, error) {
	if c.Username != "" {
		req.SetBasicAuth(c.Username, c.Password)
	} else {
		req.Header.Set("X-Redmine-API-Key", c.Key)
	}
	client := *c.HTTP
	client.Timeout = 10 * time.Minute
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return client.Do(req)
}

// UploadFile streams a local file; the returned token must be attached to an issue.
func (c *Client) UploadFile(ctx context.Context, path string, progress TransferProgress) (Upload, error) {
	var result Upload
	f, e := os.Open(path)
	if e != nil {
		return result, e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil {
		return result, e
	}
	if !info.Mode().IsRegular() {
		return result, fmt.Errorf("выберите обычный файл")
	}
	u := *c.Base
	u.Path = strings.TrimRight(u.Path, "/") + "/uploads.json"
	u.RawQuery = url.Values{"filename": {filepath.Base(path)}}.Encode()
	req, e := http.NewRequestWithContext(ctx, "POST", u.String(), &progressReader{reader: f, total: info.Size(), progress: progress})
	if e != nil {
		return result, e
	}
	req.ContentLength = info.Size()
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Accept", "application/json")
	res, e := c.transfer(req)
	if e != nil {
		return result, fmt.Errorf("загрузка файла: %w", e)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		return result, responseError(res)
	}
	var v struct {
		Upload Upload `json:"upload"`
	}
	if e = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&v); e != nil {
		return result, e
	}
	if v.Upload.Token == "" {
		return result, fmt.Errorf("сервер не вернул токен вложения")
	}
	result = v.Upload
	result.Filename = filepath.Base(path)
	result.ContentType = mime.TypeByExtension(filepath.Ext(path))
	return result, nil
}

// Download streams only from this Redmine origin. Never forward credentials to a
// foreign content_url or follow redirects supplied by a server or reverse proxy.
func (c *Client) Download(ctx context.Context, a Attachment, dst io.Writer, progress TransferProgress) error {
	u, e := url.Parse(a.ContentURL)
	if e != nil {
		return fmt.Errorf("некорректный адрес вложения")
	}
	u = c.Base.ResolveReference(u)
	if u.Scheme != c.Base.Scheme || !strings.EqualFold(u.Host, c.Base.Host) || u.User != nil || u.Fragment != "" {
		return fmt.Errorf("адрес вложения ведёт на другой сервер")
	}
	prefix := strings.TrimRight(c.Base.Path, "/") + "/attachments/"
	if !strings.HasPrefix(u.Path, prefix) {
		return fmt.Errorf("некорректный путь вложения")
	}
	req, e := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if e != nil {
		return e
	}
	res, e := c.transfer(req)
	if e != nil {
		return fmt.Errorf("скачивание файла: %w", e)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return responseError(res)
	}
	total := res.ContentLength
	if total < 0 {
		total = a.Size
	}
	_, e = io.Copy(dst, &progressReader{reader: res.Body, total: total, progress: progress})
	return e
}

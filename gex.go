package gex

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"
)

var client http.Client

type Task struct {
	url     string
	dir     string
	out     string
	header  http.Header
	timeout time.Duration
	r       Result
}

func (t *Task) URL() string {
	return t.url
}
func (t *Task) Path() string {
	return t.pathSave()
}
func (t *Task) Result() *Result {
	return &t.r
}

func (t *Task) SetTimeout(timeout time.Duration) *Task {
	t.timeout = timeout
	return t
}
func (t *Task) SetHeader(header http.Header) *Task {
	t.header = header
	return t
}
func (t *Task) SetOutputDir(dir string) *Task {
	t.dir = dir
	return t
}
func (t *Task) SetOutput(out string) *Task {
	t.out = out
	return t
}

func (t *Task) Do(ctx context.Context) (err error) {
	t.r.URL = t.url
	t.r.Path = t.pathSave()

	if t.skip() {
		return
	}

	if t.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.timeout)
		defer cancel()
	}
	t.r.TimeBegin = time.Now()
	t.r.Header, t.r.Err = t.do(ctx)
	t.r.TimeFinish = time.Now()
	if t.r.Err == nil {
		t.r.Err = t.rename()
	}
	if t.r.Err == nil {
		_ = os.Chtimes(t.pathSave(), t.r.MTime(), t.r.MTime())
	}

	return
}
func (t *Task) do(ctx context.Context) (header http.Header, err error) {
	f, stat, err := t.open()
	if err != nil {
		err = fmt.Errorf("open file failed: %w", err)
		return
	}
	defer f.Close()

	req, err := http.NewRequest(http.MethodGet, t.url, nil)
	if err != nil {
		return
	}
	req.Header.Set("range", fmt.Sprintf("bytes=%d-", stat.Size()))
	for hdr := range t.header {
		req.Header[hdr] = req.Header[hdr]
	}

	rsp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return
	}
	defer func() {
		_, _ = io.Copy(io.Discard, rsp.Body)
		_ = rsp.Body.Close()
	}()

	switch rsp.StatusCode {
	case http.StatusPartialContent:
		_, _ = f.Seek(0, io.SeekEnd)
	case http.StatusOK, http.StatusRequestedRangeNotSatisfiable:
		_ = f.Truncate(0)
	default:
		err = fmt.Errorf("invalid status code %d(%s)", rsp.StatusCode, rsp.Status)
		return
	}

	_, err = io.Copy(f, io.TeeReader(rsp.Body, &t.r.bc))
	if err != nil {
		err = fmt.Errorf("copy error: %s", err)
		return
	}
	return rsp.Header, err
}

func (t *Task) open() (f *os.File, stat os.FileInfo, err error) {
	f, err = os.OpenFile(t.pathTemp(), os.O_CREATE|os.O_WRONLY, 0766)
	if err == nil {
		stat, err = f.Stat()
	}
	return
}
func (t *Task) rename() error {
	err := os.Rename(t.pathTemp(), t.pathSave())
	if err != nil {
		err = fmt.Errorf("rename %s to %s failed: %w", t.pathTemp(), t.pathSave(), err)
	}
	if err == nil {
		t.createOk() // create .ok file
	}
	return err
}
func (t *Task) createOk() {
	f, err := os.Create(t.pathOk())
	if err == nil {
		_ = f.Close()
	}
}

func (t *Task) pathTemp() string {
	return t.pathSave() + ".dat"
}
func (t *Task) pathOk() string {
	return t.pathSave() + ".ok"
}
func (t *Task) pathSave() string {
	return filepath.Join(t.dir, t.out)
}

func (t *Task) skip() bool {
	_, err := os.Stat(t.pathOk())
	if err == nil {
		return true
	}
	return false
}
func (t *Task) hook(fn func(t *Task)) {
	if fn != nil {
		fn(t)
	}
}

func DefaultHeader() (h http.Header) {
	h = make(http.Header)
	h.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:94.0) Gecko/20100101 Firefox/94.0")
	return
}
func NewTask(uri string) *Task {
	t := &Task{url: uri}

	t.SetHeader(DefaultHeader())
	t.SetTimeout(time.Minute * 2)

	out := fmt.Sprintf("%x", md5.Sum([]byte(uri)))
	u, err := url.Parse(uri)
	if err == nil {
		out += path.Ext(u.Path)
	}
	t.SetOutput(out)

	return t
}

type bytesCounter struct {
	v int64
}

func (b *bytesCounter) Write(p []byte) (n int, err error) {
	n = len(p)
	b.v += int64(n)
	return
}

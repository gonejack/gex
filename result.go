package gex

import (
	"mime"
	"net/http"
	"time"
)

type Result struct {
	URL    string
	Path   string
	Err    error
	Header http.Header

	TimeBegin  time.Time
	TimeFinish time.Time

	bc bytesCounter
}

func (r *Result) Transfer() int64 {
	return r.bc.v
}
func (r *Result) Time() time.Duration {
	return r.TimeFinish.Sub(r.TimeBegin)
}
func (r *Result) MTime() (t time.Time) {
	t = time.Now()

	if r.Header != nil {
		parsed, err := http.ParseTime(r.Header.Get("last-modified"))
		if err == nil {
			t = parsed
		}
	}

	return
}
func (r *Result) Mime() string {
	if r.Header == nil {
		return ""
	}
	ct, _, _ := mime.ParseMediaType(r.Header.Get("content-type"))
	return ct
}
func (r *Result) Extension() (ext string) {
	ct := r.Mime()
	switch ct {
	case "":
		return
	case "application/javascript", "application/x-javascript":
		ext = ".js"
	case "image/jpeg":
		ext = ".jpg"
	case "font/opentype":
		ext = ".otf"
	default:
		exs, _ := mime.ExtensionsByType(ct)
		if len(exs) > 0 {
			ext = exs[0]
		}
	}
	return
}

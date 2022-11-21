package gex

import (
	"encoding/json"
	"mime"
	"net/http"
	"os"
	"time"
)

type Response struct {
	Request *Request
	Output  string
	Header  http.Header
	Start   time.Time `json:"-"`
	End     time.Time `json:"-"`
	Size    int64
}

func (r *Response) writeInfo(name string) error {
	f, err := os.Create(name)
	if err == nil {
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "    ")
		err = enc.Encode(r)
	}
	return err
}
func (r *Response) Mime() string {
	if r.Header == nil {
		return ""
	}
	ct, _, _ := mime.ParseMediaType(r.Header.Get("content-type"))
	return ct
}
func (r *Response) Ext() (ext string) {
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
func (r *Response) ModTime() (t time.Time) {
	t = time.Now()
	if r.Header != nil {
		parsed, err := http.ParseTime(r.Header.Get("last-modified"))
		if err == nil {
			t = parsed
		}
	}
	return
}

func DefaultHeader() (h http.Header) {
	h = make(http.Header)
	h.Set("User-Agent", "Wget/1.21.3")
	h.Set("Accept", "*/*")
	h.Set("Accept-Encoding", "identity")
	return
}

package gex

import (
	"bytes"
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

func (r *Response) saveAsJson(name string) error {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "    ")
	err := enc.Encode(r)
	if err == nil {
		err = os.WriteFile(name, buf.Bytes(), 0666)
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

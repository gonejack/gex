package gex

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

var client http.Client
var infoSuffix = ".json"

type Request struct {
	Url      string
	Header   http.Header
	Output   string
	Timeout  time.Duration `json:"-"`
	Response *Response     `json:"-"`
}

func (r *Request) Do(ctx context.Context) (err error) {
	return r.DoWithClient(ctx, &client)
}
func (r *Request) DoWithClient(ctx context.Context, client *http.Client) (err error) {
	if ctx == nil {
		ctx = context.TODO()
	}
	if r.skip() {
		r.Response, err = r.readInfo()
		return
	}

	resp := &Response{
		Request: r,
		Output:  r.Output,
		Start:   time.Now(),
	}
	r.Response = resp

	defer func() {
		resp.End = time.Now()
	}()

	file, stat, err := r.openPartial()
	if err != nil {
		return
	}
	defer file.Close()

	req, err := r.rangeRequest(stat)
	if err != nil {
		return
	}
	if r.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.Timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}
	rsp, err := client.Do(req)
	if err != nil {
		return
	}
	defer func() {
		_, _ = io.Copy(io.Discard, rsp.Body)
		_ = rsp.Body.Close()
	}()

	switch rsp.StatusCode {
	case http.StatusPartialContent:
		_, _ = file.Seek(0, io.SeekEnd)
	case http.StatusOK, http.StatusRequestedRangeNotSatisfiable:
		_ = file.Truncate(0)
	default:
		err = fmt.Errorf("invalid status code %d(%s)", rsp.StatusCode, rsp.Status)
		return
	}
	resp.Header = rsp.Header
	resp.Size, err = io.Copy(file, rsp.Body)
	if err != nil {
		return
	}
	file.Close() // for Windows
	err = os.Rename(file.Name(), r.Output)
	if err != nil {
		return
	}
	err = resp.writeInfo(r.Output + infoSuffix)

	return
}
func (r *Request) skip() bool {
	_, err := os.Stat(r.Output + infoSuffix)
	if err == nil {
		return true
	}
	return false
}
func (r *Request) readInfo() (rsp *Response, err error) {
	rsp = new(Response)
	f, err := os.Open(r.Output + infoSuffix)
	if err == nil {
		defer f.Close()
		err = json.NewDecoder(f).Decode(rsp)
	}
	return
}
func (r *Request) openPartial() (f *os.File, stat os.FileInfo, err error) {
	f, err = os.OpenFile(r.Output+".partial", os.O_RDWR|os.O_CREATE, 0766)
	if err == nil {
		stat, err = f.Stat()
	}
	return
}
func (r *Request) rangeRequest(stat os.FileInfo) (req *http.Request, err error) {
	req, err = http.NewRequest(http.MethodGet, r.Url, nil)
	if err != nil {
		return
	}

	req.Header.Set("range", fmt.Sprintf("bytes=%d-", stat.Size()))
	for hdr := range r.Header {
		req.Header[hdr] = r.Header[hdr]
	}

	return
}

func NewRequest(dir string, requrl string) (r *Request) {
	r = &Request{
		Url:     requrl,
		Timeout: time.Minute * 2,
		Header:  DefaultHeader(),
	}
	u, err := url.Parse(requrl)
	if err == nil {
		r.Output = path.Join(dir, fmt.Sprintf("%x", md5.Sum([]byte(requrl)))+path.Ext(u.Path))
	}
	return
}

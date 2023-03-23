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
	r.Response = &Response{
		Request: r,
		Output:  r.Output,
		Start:   time.Now(),
	}
	defer func() {
		r.Response.End = time.Now()
		if err == nil {
			r.Response.saveAsJson(r.Output + infoSuffix)
		}
	}()
	f, stat, err := r.openCreatePartial()
	if err != nil {
		return
	}
	defer func() {
		f.Close()
		if err == nil {
			err = os.Rename(f.Name(), r.Output)
		}
	}()
	rq, err := r.rangeRequest(stat)
	if err != nil {
		return
	}
	if r.Timeout > 0 {
		timeout, cancel := context.WithTimeout(ctx, r.Timeout)
		defer cancel()
		rq = rq.WithContext(timeout)
	}
	rp, err := client.Do(rq)
	if err == nil {
		defer func() {
			io.Copy(io.Discard, rp.Body)
			rp.Body.Close()
		}()
		switch rp.StatusCode {
		case http.StatusPartialContent:
			_, _ = f.Seek(0, io.SeekEnd)
		case http.StatusOK, http.StatusRequestedRangeNotSatisfiable:
			_ = f.Truncate(0)
		default:
			return fmt.Errorf("invalid status code %d(%s)", rp.StatusCode, rp.Status)
		}
		r.Response.Header = rp.Header
		r.Response.Size, err = io.Copy(f, rp.Body)
	}
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
func (r *Request) openCreatePartial() (f *os.File, stat os.FileInfo, err error) {
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
func (r *Request) SetHeader(k, v string) {
	r.Header.Set(k, v)
}
func (r *Request) SetTimeout(timeout time.Duration) {
	r.Timeout = timeout
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

func DefaultHeader() (h http.Header) {
	h = make(http.Header)
	h.Set("User-Agent", "Wget/1.21.3")
	h.Set("Accept", "*/*")
	h.Set("Accept-Encoding", "identity")
	return
}

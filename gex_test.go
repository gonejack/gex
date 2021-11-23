package gex_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/gonejack/gex"
)

func TestSingle(t *testing.T) {
	r := gex.NewRequest(".", "https://wx2.sinaimg.cn/large/008h3uCply1gtumw52q7aj31nj27enk8.jpg")
	r.Timeout = time.Second * 10

	err := r.Do(nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMany(t *testing.T) {
	r1 := gex.NewRequest(".", "https://wx2.sinaimg.cn/large/008h3uCply1gtumw52q7aj31nj27enk8.jpg")
	r2 := gex.NewRequest(".", "https://wx1.sinaimg.cn/large/002MwiQagy1gwmv16ypycj60dcatwe8102.jpg")

	b := gex.NewBatch(3)
	b.OnStop(func(r *gex.Request, err error) { fmt.Println(err) })
	b.Add(r1, r2)
	b.Run()
}

# gex
Goland http file download library with http range support.

[![GitHub license](https://img.shields.io/github/license/gonejack/gex.svg?color=blue)](LICENSE)

### Install
```shell
> go get github.com/gonejack/gex
```

### Usage
single task
```go
func TestSingle(t *testing.T) {
    task := gex.NewTask("https://wx2.sinaimg.cn/large/008h3uCply1gtumw52q7aj31nj27enk8.jpg")
    task.SetTimeout(time.Second * 10)
    err := task.Do(context.TODO())
    if err == nil {
        println(humanize.IBytes(uint64(task.Result().Transfer())))
        println(task.Result().Transfer())
    }
}
```

batch tasks
```go
func TestMany(t *testing.T) {
    t1 := gex.NewTask("https://wx2.sinaimg.cn/large/008h3uCply1gtumw52q7aj31nj27enk8.jpg")
    t2 := gex.NewTask("https://wx1.sinaimg.cn/large/002MwiQagy1gwmv16ypycj60dcatwe8102.jpg")
    
    var bat gex.Batch
    
    bat.Add(t1, t2)
    bat.Run()
    
    t.Log(t1.Result())
    t.Log(t2.Result().Err)
    t.Log(t2.Result().Time())
}
```

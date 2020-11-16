# Context
上下文 context.Context 是用来设置截止日期、同步信号，传递请求相关值的结构体。上下文与 Goroutine 有比较密切的关系。context.Context 是 Go 语言中独特的设计，在其他编程语言中我们很少见到类似的概念。

context.Context 是 Go 语言在 1.7 版本中引入标准库的接口1，该接口定义了四个需要实现的方法，其中包括：

1. Deadline — 返回 context.Context 被取消的时间，也就是完成工作的截止日期；
2. Done — 返回一个 Channel，这个 Channel 会在当前工作完成或者上下文被取消之后关闭，多次调用 Done 方法会返回同一个 Channel；
3. Err — 返回 context.Context 结束的原因，它只会在 Done 返回的 Channel 被关闭时才会返回非空的值；
    * 如果 context.Context 被取消，会返回 Canceled 错误；
    * 如果 context.Context 超时，会返回 DeadlineExceeded 错误；
4. Value — 从 context.Context 中获取键对应的值，对于同一个上下文来说，多次调用 Value 并传入相同的 Key 会返回相同的结果，该方法可以用来传递请求特定的数据；

```
type Context interface {
	Deadline() (deadline time.Time, ok bool)
	Done() <-chan struct{}
	Err() error
	Value(key interface{}) interface{}
}
```
context 包中提供的 context.Background、context.TODO、context.WithDeadline 和 context.WithValue 函数会返回实现该接口的私有结构体。

## 默认上下文
context包中最常用的方法还是context.Backgroup、context.TODO，这两个方法都会返回预先初始化好的私有变量 backgroud和todo，他们会在同一个Go携程被复用：
```
func Backgroud() Context {
    return backgroud
}

func TODO() Context {
    return todo
}
```
### Context层级关系
从源代码来看，context.Background 和 context.TODO 函数其实也只是互为别名，没有太大的差别。它们只是在使用和语义上稍有不同：
* context.Backgroud 是上下文的默认值，所有其他的上下文都应该从它衍生出来。
* context.TODO 应该只在不确定使用哪种上下文时使用。

## 取消信号
context.WithCancel 函数能够从context.Context中衍生出一个新的子上下文并返回用于取消该上下文的函数（CancelFunc）。一旦我们执行返回的取消函数，当前上下文以及它的子上下文都会被取消，所有的 Goroutine 都会同步收到这一取消信号。




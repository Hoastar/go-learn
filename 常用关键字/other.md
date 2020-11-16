# panic 和 recover
## 功能介绍
panic 和 recover，这两个关键字都与 defer 有千丝万缕的联系，也都是 Go 语言中的内置函数，但是提供的功能却是互补的：
* panic 能够改变程序的控制流，函数调用panic 时会立刻停止执行函数的其他代码，并在执行结束后在当前 Goroutine 中递归执行调用方的延迟函数调用 defer；
* recover 可以中止 panic 造成的程序崩溃。它是一个只能在 defer 中发挥作用的函数，在其他作用域中调用不会发挥任何作用；

## 现象
* panic 只会触发当前 Goroutine 的延迟函数调用；
* recover 只有在 panic后的 defer 函数中调用才会生效；
* panic 允许在 defer 中嵌套多次调用；


## 数据结构
panic关键字在Go语言的源代码是由数据结构runtime._panic表示的。没当我们调用panic都会创建一个如下所示的数据结构存储相关信息：
```
type _panic struct {
    argp unsafe.Pointer
    arg interface{}
    link *_panic
    recovered bool
    aborted bool

    pc uintptr
    sp unsafe.Pointer
    goexit bool
}
```

1. argp 是指向 defer 调用时参数的指针；
2. arg 是调用 panic 时传入的参数；
3. link 指向了更早调用的 runtime._panic 结构；
4. recovered 表示当前 runtime._panic 是否被 recover 恢复；
5. aborted 表示当前的 panic 是否被强行终止；

## 程序崩溃
 panic 函数是如何终止程序的？ 编译器会将关键字 panic 转换成 runtime.gopanic，该函数的执行过程包含以下几个步骤：

 1. 创建新的 runtime._panic 结构并添加到所在的Goroutine _panic链表的最前面。
 2. 在循环中不断从当前 Goroutine 的 _defer 中链表获取 runtime._defer 并调用 runtime.reflectcall 运行延迟调用函数
 3. 调用 runtime.fatalpanic 中止整个程序
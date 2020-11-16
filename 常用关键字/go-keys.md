# go 关键字
## defer
现代很多编程语言中都有defer关键字，Go语言的defer会在当前函数或者方法返回之前执行传入的函数。它常被用来关闭文件描述符、关闭数据库连接以及解锁资源。

作为一个编程语言的关键字，defer的实现一定是由编译器和运行时共同完成的。

使用defer的最常见的场景就是在函数调用结束后，完成u一些收尾的工作，例如在defer中回滚数据库的事物：

```
func createPost(db *gorm.DB) error {
    tx := db.Begin()
    defer tx.Rollback()
    
    if err := tx.Create(&Post{Author: "Draveness"}).Error; err != nil {
        return err
    }
    
    return tx.Commit().Error
}
```

在使用数据库事务时，我们可以使用如上所示的代码在创建事务之后就立刻调用 Rollback 保证事务一定会回滚。哪怕事务真的执行成功了，那么调用 tx.Commit() 之后再执行 tx.Rollback() 也不会影响已经提交的事务

### 现象
我们在Go语言中使用defer是会u遇到两个比较常见的问题。

* defer关键字的调用时机以及多次调用defer时执行的顺序如何确定
* defer关键字使用传值的方式传递参数时会进行预计算，导致，不符合预期的结果；

#### 作用域

向defer关键字传入的函数会在函数返回之前运行。假设我们在for循环中多次调用defer关键字：
```
func main() {
	for i := 0; i < 5; i++ {
		defer fmt.Println(i)
	}
}

% go run main.go
4
3
2
1
0
```
运行上述代码会倒序执行所有向defer关键字中传入的表达式，最后一次defer调用传入了fmt.Println(4)，所以这段代码会优先打印4。

```
func main() {
    {
        defer fmt.Println("defer runs")
        fmt.Println("block ends")
    }
    
    fmt.Println("main ends")
}

 % go run main.go 
block ends
mian ends
defer runs
```
从上述代码的输出我们就会发现，defer传入的函数不是在退出代码块的作用域时执行的，它只会在当前的函数和方法返回之前被调用。

### 预计算参数
Go语言中所有的函数调用都是传值的，defer虽然时关键字，但也继承了这个特性。假设我们想要计算main函数的运行的时间，可能会写出以下代码：

```
func main() {
	startedAt := time.Now()
	defer fmt.Println(time.Since(startedAt))
	
	time.Sleep(time.Second)
}

% go run main.go
0s
```
然而上述代码的结果并不符合我们的预期，经过分析，我们会发现调用 defer 关键字会立刻对函数中引用的外部参数进行拷贝，所以time.Since(startedAt)的结果不是在main函数退出之前计算的，而是在 defer 关键字调用时计算的，最终导致上述代码输出 0s。

想要解决这个问题的方法非常简单，我们只需要向 defer 关键字传入匿名函数：
```
func main() {
	startedAt := time.Now()
	defer func() { fmt.Println(time.Since(startedAt)) }()
	
	time.Sleep(time.Second)
}

% go run main.go
1s
```

虽然调用 defer 关键字时也使用值传递，但是因为拷贝的是函数指针，所以 time.Since(startedAt) 会在 main 函数返回前被调用并打印出符合预期的结果。
### 数据结构
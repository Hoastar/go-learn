# go关键字 defer
## 介绍
Go 语言的 defer（或defer代码块） 会在当前函数或者方法返回之前执行传入的函数（在函数调用链表中增加一个函数调用）。它会经常被用于关闭文件描述符、关闭数据库连接以及解锁资源。 

Go 语言中使用 defer 时会遇到两个比较常见的问题
* defer 关键字的调用时机以及多次调用 defer 时执行顺序是如何确定的；也就是说defer只对当前协程有效（main可以看作是主协程）；
* defer 关键字使用传值的方式传递参数时会进行预计算，导致不符合预期的结果；

作用域：
1. 向 defer 关键字传入的函数会在函数返回之前运行。假设我们在 for 循环中多次调用 defer 关键字， 我们会发现，defer 传入的函数不是在退出代码块的作用域时执行的，它只会在当前函数和方法返回之前被调用。

其中return其实应该包含前后两个步骤：第一步是给返回值赋值（若为有名返回值则直接赋值，若为匿名返回值则先声明再赋值）；第二步是调用RET返回指令并传入返回值，而RET则会检查defer是否存在，若存在就先逆序插播defer语句，最后RET携带返回值退出函数。

预计算参数：
1. defer声明时会先计算确定参数的值，defer推迟执行的仅是其函数体。


规则：
1. 当defer被声明时，其参数就会被实时解析
```
func a() {
	i := 0
	defer fmt.Println(i)
	i++
	return
}

% go run main.go
0
```
    以上代码块运行结果为0。这是因为：虽然我们在defer后面定义的是一个带变量的函数：fmt.Println(i),但这个变量在defer被声明的时候，就已经确定其确定的值了。
    换言之，以上代码等同于以下：
```
func a() {
	i := 0
	defer fmt.Println(0) //因为i=0，所以此时就明确告诉golang在程序退出时，执行输出0的操作
	i++
	return
}
```
    通过运行结果，可以看到defer输出的值，就是定义时的值。而不是defer真正执行时变量值。


```
func f1() (result int) {
	defer func() {
		result++
	}()
	return 0
}

/*
func f11() (result int) {
    result = 0  //先给返回值赋值(默认值)
    func(){     //再执行defer 函数
        result++
    }()
    return      //最后返回
}
*/

func f2() (r int) {
	t := 5
	defer func() {
		t = t+5
	}()
	return t
}

func f3() (t int) {
	t = 5
	defer func() {
		t = t+5
	}()
	return t
}
func f4() (r int) {
	defer func(r int) {
		r = r + 5
	}(r)
	return 1
}

func main() {
	fmt.Println(f1())
	fmt.Println(f2())
	fmt.Println(f3())
	fmt.Println(f4())
}

% go run main.go
1
5
10
1
```
    函数返回过程：先给返回值赋值，然后调用defer表达式，最后才是返回到调用函数中。
    defer表达式可能会在设置函数返回值之后，在返回到调用函数之前，修改返回值，使最终的函数返回值与你想象的不一致。
2. 多个defer执行顺序为先进后出（后进先出）。
3. defer可以读取有名返回值
匿名返回值是在return执行时被声明，有名返回值则是在函数声明的同时被声明，因此在defer语句中只能访问有名返回值，而不能直接访问匿名返回值。


### 编译过程
### 运行过程
defer关键字的运行时分实现分为两个过程：
* runtime.deferproc 函数负责创建新的延迟调用
* runtime.deferreturn 函数负责在函数调用结束时执行所有的延长调用。


#### 创建延迟调用
runtime.deferproc 会为defer创建一个新的runtime._defer结构体元素，它都会被追加到所在的 Goroutine _defer 链表的最前面（关键字插入时是从后向前的），defer 关键字接着从前访问执行（而这就是后调用的 defer 会优先执行的原因）。
#### 
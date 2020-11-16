# 函数调用
函数是Go语言的一等公民，可以从函数的调用惯例和参数的传递方法两个方面介绍函数的执行过程。
## 调用惯例
无论是系统级编程语言C和Go,还是脚本语言Ruby和Python，这些在语言在调用函数时往往都使用相同的语法：
```
testfunction(arg0, arg1)
```
虽它们调用函数的语法类似，但调用惯例却可能大不相同。调用惯例是调用方和被调用方对于参数和返回值传递的约定。
### Go语言main函数的调用栈
通过分析 Go 语言编译后的汇编指令，我们发现 Go 语言使用栈传递参数和接收返回值，所以它只需要在栈上多分配一些内存就可以返回多个值。
C 语言和 Go 语言在设计函数的调用惯例时选择也不同的实现。C 语言同时使用寄存器和栈传递参数，使用 eax 寄存器传递返回值；而 Go 语言使用栈传递参数和返回值。
## 参数传递
* 传值： 函数调用时会对参数进行拷贝，被调用方和和被调用放两者持有不相关的两份数据；
* 传引用：函数调用时会传递参数的指针，被调用方和调用方两者持有相同的数据，任意一方做出的修改都会影响另一方。

不同语言会选择不同的方式传递参数，Go 语言选择了传值的方式，无论是传递基本类型、结构体还是指针，都会对传递的参数进行拷贝。

#### 整型和数组
```
func myFunction(i int, arr [2]int) {
	fmt.Printf("in my_funciton - i=(%d, %p) arr=(%v, %p)\n", i, &i, arr, &arr)
}

func main() {
	i := 30
	arr := [2]int{66, 77}
	fmt.Printf("before calling - i=(%d, %p) arr=(%v, %p)\n", i, &i, arr, &arr)
	myFunction(i, arr)
	fmt.Printf("after  calling - i=(%d, %p) arr=(%v, %p)\n", i, &i, arr, &arr)
}

% go run main.go
before calling - i=(30, 0xc00009a000) arr=([66 77], 0xc00009a010)
in my_funciton - i=(30, 0xc00009a008) arr=([66 77], 0xc00009a020)
after  calling - i=(30, 0xc00009a000) arr=([66 77], 0xc00009a010)

```
结论： Go 语言中对于整型和数组类型的参数都是值传递的

#### 结构体和指针
```
type MyStruct struct {
	i int
}

func myFunction(a MyStruct, b *MyStruct) {
	a.i = 31
	b.i = 41
	fmt.Printf("in my_function - a=(%d, %p) b=(%v, %p)\n", a, &a, b, &b)
}

func main() {
	a := MyStruct{i: 30}
	b := &MyStruct{i: 40}
	fmt.Printf("before calling - a=(%d, %p) b=(%v, %p)\n", a, &a, b, &b)
	myFunction(a, b)
	fmt.Printf("after calling  - a=(%d, %p) b=(%v, %p)\n", a, &a, b, &b)
}

% go run main.go
before calling - a=({30}, 0xc0000140f0) b=(&{40}, 0xc00000e028)
in my_function - a=({31}, 0xc000014110) b=(&{41}, 0xc00000e038)
after calling  - a=({30}, 0xc0000140f0) b=(&{41}, 0xc00000e028)
```
结论：
* 传递结构体时：会对结构体中的全部内容进行拷贝；
* 传递结构体指针时：会对结构体指针进行拷贝；
* 将指针作为参数传入某一个函数时，在函数内部会对指针进行复制，也就是会同时出现两个指针指向原有的内存空间，所以 Go 语言中『传指针』也是传值。


### 小结
Go 语言的调用惯例，包括传递参数和返回值的过程和原理。Go 通过栈传递函数的参数和返回值，在调用函数之前会在栈上为返回值分配合适的内存空间，随后将入参从右到左按顺序压栈并拷贝参数，返回值会被存储到调用方预留好的栈空间上，我们可以简单总结出以下几条规则：

* 通过堆栈传递参数，入栈的顺序是从右到左；
* 函数返回值通过堆栈传递并由调用者预先分配内存空间；
* 调用函数时都是传值，接收方会对入参进行复制再计算；
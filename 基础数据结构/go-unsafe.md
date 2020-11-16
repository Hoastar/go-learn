# golang unsafe
## 指针类型
golang 是一种强类型的语言，golang 的指针多了一些限制。这是Golang 的成功之处：既可以享受指针带来的便利，又避免了指针的危险性。

1. 指针是不能做数学运算的
在比如一下语言（c++等），我们如果想通过数组指针访问不同的索引的元素，直接用 ptr++ 之类的操作就可以，但是在 golang里面 对指针的任何数学运算都是不被允许的。

2. 不同类型的指针不能相关转换。
也就是说一个指针只能指向一种数据类型，即使是golang里面的组合关系，或者是同样像是某个interface的两个struct 也不能相关转换。

3. 不同类型的指针不能使用 == 或者 != 比较
4. 不同类型的指针变量不能相互赋值

## unsafe内容介绍

unsafe包提供了访问底层内存的方法。是用unsafe函数可以提高访问对象的速度。

golang里面的指针是类型安全的，是因为编译器帮我们做了跟多检查，但这必然带来性能损失。对于高阶程序员有非安全的指针，这就是unsafe包提供的 unsafe.Pointer，在某些情况下，它会使代码更加高效，也然也伴随这危险。

```
type ArbitraryType int
type Pointer *ArbitraryType


// Sizeof 返回（类型）变量在内存中占用的字节数，切记，如果是slice，为 slice header 的大小。不会返回这个slice在内存中的实际占用长度（字节）。
func Sizeof(v ArbitraryType) uintptr 

// Alignof 返回变量对齐字节数量。Alignof 返回 m，m 是指当类型进行内存对齐时，它分配到的内存地址能整除 m。
func Alignof(v ArbitraryType) uintptr 

// Offsetof返回变量指定属性的偏移量（结构体成员在内存中的位置离结构体起始处的字节数，所传参数必须是结构体的成员），这个函数虽然接收的是任何类型的变量，但是这个又一个前提，就是变量要是一个struct类型，且还不能直接将这个struct类型的变量当作参数，只能将这个struct类型变量的属性当作参数。
func Offsetof(v ArbitraryType) uintptr 
```

* 通过指针加偏移量的操作，在地址中，修改，访问变量的值
在这个包中，只提供了三个函数，两个类型

unsafe中，unsafe中，通过这两个个兼容万物的类型，将其他类型都转换过来，然后通过这三个函数，分别能取长度，偏移量，对齐字节数，就可以在内存地址映射中，来回游走。

1. uintptr：Go 的内置类型，返回无符号整数，可存储一个完整的地址，后续常用于指针数学运算。GC 不把 uintptr 当指针，uintptr 无法持有对象，uintptr 类型的目标会被回收。
2. Pointer：表示指向任意类型的指针。可以保护它所指向的对象在“有用”的时候不会被垃圾回收。

unsafe.Pointer 可以和普通指针相互转换。
unsafe.Pointer 可以和 uintptr 进行相互转换。
也可以这么理解， unsafe.Pointer 是桥梁，可以让任意类型的指针实现相互转换，也可以将任意类型的指针转换为 uintptr 进行指针运算。

详细说明

* type ArbitraryType int

ArbitraryType 是int的一个别名，但是golang中，对ArbitraryType赋予了特殊的意义

* type Pointer *ArbitraryType

Pointer 是int指针类型的一个别名，在golang系统中，可以把Pointer类型，理解成任何指针的父亲。



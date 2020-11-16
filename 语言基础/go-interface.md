# 接口
## 概述
Go 语言中的接口是一种内置的类型，Go 语言中的接口就是一组方法的签名，是golang的重要组成部分。

### 隐式接口
Go语言中定义接口需要使用 interface 关键字，并且在接口中只能定义方法签名，不包含成员变量。
```
type error interface {
    Error() string
}
```
如果需要时现上述 error 接口，那么它只需要实现 Error() string 方法，下面的 RPCError结构体就是 error 接口的一个实现。
```
type PRCError struct {
    Code int64
    Message string
}

func (e *RPCError) Error() string {
    return fmt.Sprintf("%s, code=%d", e.Message, e.Code)
}
```
Go 语言中接口的实现都是隐式的，我们只需要实现 Error() string 方法实现了 error 接口。Go 语言实现接口的方式与 Java 完全不同：

* 在 Java 中：实现接口需要显式的声明接口并实现所有方法；
* 在 Go 中：实现接口的所有方法就隐式的实现了接口；

### 两种接口
接口也是 Go 语言中的一种类型，它能够出现在变量的定义、函数的入参和返回值中并对它们进行约束，不过 Go 语言中有两种略微不同的接口，一种是带有一组方法的接口，另一种是不带任何方法的 interface{}：
Go语言使用 iface 结构体表示第一种接口，使用 eface 结构体表示第二种空接口，两种接口虽然都使用 interface 声明，但是由于后者在 Go 语言中非常常见，所以在实现时使用了特殊的类型。
```
package main
 
func main() {
    type Test struct{}
    v := Test{}
    Print(v)
}

func Print(v interface{}) {
    fmt.Println(v)
}
```
上述函数不接受任意类型的参数，只接受 interface{} 类型的值，在调用 Print 函数时会对参数 v 进行类型转换，将原来的 Test 类型转换成 interface{} 类型。

### 指针与接口
在 Go 语言中同时使用指针和接口时会发生一些让人困惑的问题，接口在定义一组方法时没有对实现的接收者做限制，所以我们会看到『一个类型』实现接口的两种方式：
example 1
```
type Duck interface {
    Walk()
    Quack()
}

type Cat struct{}

func (c *Cat) Walk {
    fmt.Println("catwalk")
}

func (c *Cat) Quack {
    fmt.Println("meow")
}
```

example 2
```
type Duck interface {
    Walk()
    Quack()
}

type Cat struct{}

func (c Cat) Walk {
    fmt.Println("catwalk")
}

func (c Cat) Quack {
    fmt.Println("meow")
}
```

#### 结构体和指针实现接口
这是因为结构体类型和指针类型是完全不同的，就像我们不能向一个接受指针的函数传递结构体，在实现接口时这两种类型也不能划等号。但是上图中的两种实现不可以同时存在，Go 语言的编译器会在结构体类型和指针类型都实现一个方法时报错 —— method redeclared。

但是对于 Cat 结构体来说，它可以在实现接口时选择接受着的类型，即结构体或者结构体指针，在初始化时也可以初始化成结构体或者指针。下面的代码总结了如何实现结构体、结构体指针实现接口，以及使用结构体、结构体指针初始化变量。
```
type Cat struct {}
type Duck interface {
    Quack()
}

func (c Cat) Quack{}    // 使用结构体实现接口
func (c *Cat) Quack{}   // 使用结构体指针实现接口

var d Duck = Cat{}      // 使用结构体初始化变量
var d Duck = &Cat{}     // 使用结构体指针初始化变量
```

#### 实现接口的接受者类型
实现接口的类型和初始化返回的类型两个维度组成了四种情况，这四种情况并不都能通过编译器的检查：

* 1.方法接受者，初始化类型都是结构体。
* 2.方法接受者，初始化类型都是结构体指针。
* 3.方法接受者是结构体，初始化类型是结构体指针。
* 4.方法接受者是结构体指针，初始化类型是结构体。

-|结构体实现接口（方法接受者是结构体）|结构体指针实现接口（方法接受者是结构体指针）|
-|-|-|
结构体初始化变量|通过|不通过|
结构体指针初始化变量|通过|通过|

四种中只有『使用指针实现接口，使用结构体初始化变量』无法通过编译，其他的三种情况都可以正常执行。当实现接口的类型和初始化变量时返回的类型时相同时，代码通过编译是理所应当的：

```
type Duck interface {
	Quack()
}

type Cat struct{}

func (c *Cat) Quack() {
	fmt.Println("meow")
}

func main() {
	var c Duck = Cat{}
	c.Quack()
}

```

##### nil 和 non-nil
如何理解 Go 语言的接口类型不是任意类型：
下面的代码在 main函数中初始化了一个 *TestStruct结构体指针，由于指针的零值是 nil, 所以变量 s在初始化之后也是 nil:

```
package main

type TestStruct struct{}

func NilOrNot(v interface{}) bool {
	return v == nil
}

func main() {
	var s *TestStruct
	fmt.Println(s == nil)      // #=> true
	fmt.Println(NilOrNot(s))   // #=> false
}

% go run main.go
true
false
```
总结：
* 将上述变量与 nil 比较会返回 true；
* 将上述变量传入 NilOrNot 方法并与 nil 比较会返回 false；

原因：调用 NilOrNot 函数时发生了隐式的类型转换，除了向方法传入参数之外，变量的赋值也会触发隐式类型转换。在类型转换时，*TestStruct 类型会转换成 interface{} 类型，转换后的变量不仅包含转换前的变量，还包含变量的类型信息 TestStruct，所以转换后的变量与 nil 不相等。

### 数据结构
Go 语言根据接口类型『是否包含一组方法』对类型做了不同的处理。

* iface（struct）：包含方法
* eface（struct）：不包含方法
```
type eface struct { //16 byes
    _type *_type    // 类型
    data unsafe.Pointer    // 底层数据
}

type iface struct { // 16 bytes
	tab  *itab
	data unsafe.Pointer
}
```

#### 类型结构体：
_type 是Go语言类型的运行时表示。下面是运行时包中的结构体，结构体包含了很多元信息，例如：类型的大小、哈希、对齐以及种类等。

```
type _type struct {
	size       uintptr
	ptrdata    uintptr
	hash       uint32
	tflag      tflag
	align      uint8
	fieldAlign uint8
	kind       uint8
	equal      func(unsafe.Pointer, unsafe.Pointer) bool
	gcdata     *byte
	str        nameOff
	ptrToThis  typeOff
}
```
* size 字段存储了类型占用的内存空间，为内存分配提供信息
* hash 字段能够帮助我们快速确定类型是否相等
* equal 字段用于判断当前类型的多个对象是否相等，该字段是为了减少 Go 语言二进制包大小从 typeAlg 结构体中迁移过来的

#### itab 结构体

itab 结构体是接口类型的核心组成部分，没一个itab都占32字节的空间，可以将其看出是接口类型和具体类型的结合。他们分别用inter和_type两个字段表示：
```
type itab struct { // 32 bytes
	inter *interfacetype
	_type *_type
	hash  uint32
	_     [4]byte
	fun   [1]uintptr
}
```
其他字段也有自己的作用：
* hash 是对 _type.hash的拷贝，当我们想将 interface 类型转换成具体类型时，可以使用该字段快速判断目标类型和具体类型_type 是否一致。
* fun 是一个动态大小的数组，它是一个用于动态派发的虚函数表，存储了一组函数指针。虽然该变量被声明成大小固定的数组，但是在使用时会通过原始指针获取其中的数据，所以 fun 数组中保存的元素数量是不确定的；

### 类型转换
接口类型的初始化与传递
### 类型断言
现在介绍的是如何将一个接口类型转换成具体类型。很据接口中是否存在方法分两种情况介绍类型断言的执行过程。
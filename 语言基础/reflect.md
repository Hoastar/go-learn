# 反射（reflect）
反射是Go语言比较重要的特性。虽然在绝大多数的应用和服务中并不常见，但是很多框架都依赖Go语言的反射机制实现简化代码的逻辑。其中Go语言的语法元素很少，设计简单。Go语言的reflect包可以弥补在语法上的优势。

典型用法是用静态类型interface{}保存一个值，通过调用TypeOf获取其动态类型信息，该函数返回一个Type类型值。调用ValueOf函数返回一个Value类型值，该值代表运行时的数据。Zero接受一个Type类型参数并返回一个代表该类型零值的Value类型值。

reflect实现了运行时的反射能力，能够让程序操作不同类型的对象。反射包中有两对非常重要的函数和类型，reflect.TypeOf 能获取类型信息，reflect.ValueOf 能获取数据的运行时表示，另外两个类型是 Type 和 Value，它们与函数是一一对应的关系：

反射函数和类型如下：
Typeof --> Type
Valueof--> Value

类型Type是反射包中定义的一个接口，我们可以使用reflect.Typeof函数获取任意变量的类型，Type接口中定义了一些有趣的方法，MethodByName可以获取当前类型对应方法的引用，Implements可以判断当前类型是否实现了某个接口：
```
type Type interface {
    Align() int
    FieldAlign int
    Method(int) Method
    MethodByName(string) (Method, bool)
    NumMethod() int
    ...
    Implements(u Type) bool
}
```
反射包中 Value的类型与Type不同，Value 被声明成了结构体。这个结构体没有对外暴露的字段，但是提供了获取或者写入数据的方法：

```
type Value struct {
    //contains filtered or unexported fields 包涵未暴露的或者未过滤的字段
}

func(v Value) Addr() Value
func(v Value) Bool() bool
func(v Value) Bytes() []byte
...
```
反射包中的所有方法基本上都是围绕Type和Value这两个类型设计的。我们通过reflect.TypeOf、reflect.ValueOf可以将一个普通的变量转换成『反射』包中提供的 Type 和 Value，随后就可以使用反射包中的方法对它们进行复杂的操作。
## 三大法则
运行时反射是程序在运行期间检查其自身结构的一种方式。反射带来的灵活性是一把双刃剑，反射作为一种元编程方式可以减少重复代码，但是过量的使用反射会使我们的程序变的难以理解并且运行缓慢。
以下会介绍Go语言反射的三大法则：
1. 从 interface{} 变量可以反射出反射对象
2. 从反射对象可以获取interface{}变量
3. 要修改反射函数，其值必须可以设置。

### 第一法则
反射的第一法则是我们能将 Go 语言的 interface{} 变量转换成反射对象。疑惑的是， 为什么是从interface{}变量到反射对象？当我们执行reflect.ValueOf(1)时，虽然是获取了基本类型int对应的反射类型，但是由于reflect.TypeOf、reflect.ValueOf两个方法的入参都是 interface{}类型，所以在方法执行的过程中发生了类型转换。

在函数调用一节说过，Go 语言的函数调用都是值传递的，变量会在函数调用时进行类型转换。基本类型 int 会转换成 interface{} 类型，这也就是为什么第一条法则是『从接口到反射对象』。

上面提到的 reflect.TypeOf 和 reflect.ValueOf 函数就能完成这里的转换，如果我们认为 Go 语言的类型和反射类型处于两个不同的『世界』，那么这两个函数就是连接这两个世界的桥梁。

通过以下例子简单了解下这两个函数的作用，reflect.TypeOf获取了变量 author的类型，reflect.ValueOf获取了变量的值draven。那如果我们知道了一个变量的类型和值，那么就意味着知道了这个变量的全部信息。
```
package main

import (
    "fmt"
    "reflect"
)

func main() {
    author := "draven"
    fmt.Println("TypeOf author:", reflect.TypeOf(author))
    fmt.Println("ValueOf author:", reflect.ValueOf(author))
}
```
然而有了变量的类型之后，我们就可以通过Method方法获得类型实现的方法。通过 Field 获取类型包含的全部字段。对于不同的类型，我们也可以调用不同的方法获取相关信息：

* 结构体：获取字段的数量并通过下标和字段名获取字段 StructField；
* 哈希表：获取哈希表的 Key 类型；
* 函数或方法：获取入参和返回值的类型；
* …

总而言之，使用 reflect.TypeOf 和 reflect.ValueOf 能够获取 Go 语言中的变量对应的反射对象。一旦获取了反射对象，我们就能得到跟当前类型相关数据和操作，并可以使用这些运行时获取的结构执行方法。

### 第二法则

反射的第二法则是我们可以从反射对象可以获取 interface{} 变量。既然能够将接口类型的变量转换成反射对象，那么一定需要其他方法将反射对象还原成接口类型的变量，reflect 中的 reflect.Value.Interface 方法就能完成这项工作：
 
Interface Value <-- reflect.Value.Interface <-- Reflection Object

不过调用 reflect.Value.Interface方法只能获得interface{} 类型的变量， 如果想要将其还原成最原始的状态，还需要经过如下所示的显式类型转换：
```
v := reflect.ValueOf(1)
v.Interface().(int)
```
从反射对象到接口值的过程就是从接口值到反射对象的镜面过程，两个过程都需要经历两次转化:
* 从接口值到反射对象：
    * 从基本类型到接口类型的转换
    * 从接口类型到反射对象的转换
* 从反射对象到接口值：
    * 反射对象转换成接口类型
    * 通过显式类型转换成原始类型

当然不是所有的变量都需要类型转换这一过程。如果变量本身就是 interface{} 类型，那么它不需要类型转换，因为类型转换这一过程一般都是隐式的，所以我不太需要关心它，只有在我们需要将反射对象转换回基本类型时才需要显式的转换操作。

### 第三法则


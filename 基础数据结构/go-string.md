# 字符串
## 基本数据类型-介绍
字符串虽然在 Go 语言中是基本类型 string， 但它实际上是由字符组成的数组。Go 语言中的字符串其实是一个只读的字节数组
## 内存中的字符串
如果是代码中存在的字符串，会在编译期间被标记成只读数据 SRODATA 符号，假设我们有以下的一段代码，其中包含了一个字符串，当我们将这段代码编译成汇编语言时，就能够看到 hello 字符串有一个 SRODATA 的标记：
```
$ cat main.go
package main

func main() {
	str := "hello"
	println([]byte(str))
}

$ GOOS=linux GOARCH=amd64 go tool compile -S main.go
go.string."hello" SRODATA dupok size=5
	0x0000 68 65 6c 6c 6f                                   hello 

```
只读只意味着字符串会被分配到只读的内存空间并且这块不会被修改，但是运行时我们其实可以将这段内存拷贝到对或者栈上，将变量的类型转换成[]byte之后就可以进行修改。修改之后通过类型转换就可以变回string,Go语言是不直接支持修改string类型变量的内存空间。
## 数据结构
字符串在Go语言中的接口其实非常简单，没一个字符串在运行时都会使用如下的 StringHeader 结构体表示，在运行时包的内部其实有一个私有的结构 stringHeader, 它有着完全相同的结构只是用于存储数据的Data字段使用了 unsafe.Pointer （一个可以指向任意类型的指针）类型。
```
type StringHeader struct {
    Data uintptr  //一个足够大的无符号整型， 用来表示任意地址
    Len int
}

type stringHeader struct {
    Data  unsafe.Pointer // 一个可以指向任意类型的指针类型
    Len int
}
```
经常说字符串是一个只读的切片类型，这是因为切片在Go语言的运行时表示与字符串高度相似：
```
type SliceHeader struct {
    Data uintptr
    Len int
    Cap int
}
```
相比之下，字符串少了一个表示容量的Cap字段，因为字符串作为只读的类型，我们并不会直接向字符串直接追加元素来改变其本身的内存空间，所有字符串上执行的写入操作都是通过拷贝实现的。

## 解析过程
## 字符拼接
## 类型转换
一般，我们都会使用Go语言解析和序列化 JSON等数据格式时，经常需要将string和[]byte直接来回转换，然而类型转换的开销并没有想象中的那么小。
### 字符串和字节数组的转换
字符串和[]byte 中的内容虽然一样，但是字符串内容是只读的，我们不能通过下标（索引）或者其他形式改变其中的数据没，而[]byte的内容是可以读写的，无论从那种类型转换到另一种都需要对其中的内容进行copy。而内存拷贝的性能损耗会随着字符串和 []byte 长度的增长而增长。
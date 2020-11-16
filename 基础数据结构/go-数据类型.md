# 数据类型
## golang数据类型：
1. 布尔
2. 数字类型
3. 字符串类型
4. 派生类型:
    * (a) 指针类型（Pointer）
    * (b) 数组类型
    * (c) 结构体类型(struct)
    * (d) Channel 类型
    * (e) 函数类型 
    * (f) 切片类型 
    * (g) 接口类型（interface）
    * (h) Map 类型

## 划分

大致将以上go数据类型按值类型与引用类型分为两部分：

Value Types：
* int
* float
* string
* bool
* structs

Reference Types：
* slices
* maps
* channels
* pointers
* functions

两者区别：
* 值类型：内存中变量存储的是具体的值，内存通常在栈中分配。
* 引用类型：变量直接存储的是一个内存地址值，这个内存地址值指向的内存空间存放的才是值。


 
[ps](http://c.biancheng.net/view/18.html):
* uint8 类型，或者叫 byte 型，byte 类型是 uint8 的别名，代表了 ASCII 码的一个字符。
* rune 类型，代表一个 UTF-8 字符，rune 类型等价于 int32 类型
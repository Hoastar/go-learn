## 函数
----
### 语法介绍
    
    func 函数名 (形参列表) (返回值列表) {
        执行语句
        return 返回值列表
    }
    
### 函数调用
#### 递归调用
#### 注意事项
* 12.使用下划线_标识符，忽略返回值。
* 13.支持可变参数
    * args 是slice切片，通过 arg[index] ，可以访问到各个值。
    * 如果一个函数的形参列表中有可变参数，则可变参数需要放在确定的形参列表之后。

#### init函数
* 基本介绍:
    每一个源文件都可以包含一个init函数，该函数会在main函数执行前，被go运行框架调用，也就是说init会在main函数前被调用。

* 细节介绍:
    * 如果一个文件同时包含全局变量定义，init函数和main函数，则执行的流程为 全局变量定义-> init函数-> main函数

#### 匿名函数
* 介绍:
    * go支持匿名函数，如果我们希望只是使用一次，可以使用匿名函数，当然匿名函数也可以实现多次调用

* 使用方式:
    * 在定义时就是直接调用，这种匿名函数只能调用一次
    * 将匿名函数赋值给一个变量(函数变量)，在通过该变量来调用匿名函数
#### 闭包
#### defer
    * 介绍: 在函数中，程序员经常需要创建资源，比如数据库连接，文件句柄，锁等。为了在函数执行完毕后，及时的释放资源，Go提供defer(延时机制)

#### 函数参数的传递方式
    * 介绍: 包括值类型和引用类型，值类型参数默认就是值传递，而引用类型参数默认就是引用传递。
        * 值传递
        * 引用传递
    其实，不管是值传递还是引用传递给函数的都是被量的副本，不同的是值传递是值的拷贝，引用传递是地址的拷贝，一般来说，地址拷贝效率高，因为数据量小，而值拷贝由数据大小决定，数据越大，效率越低。

#### 变量作用域

#### 字符串相关函数

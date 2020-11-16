# Channel
本节主要学习管道 Channel的设计原理、数据结构、和常见操作。例如 Channel的创建、发送、接受和关闭。

## 设计原理
作为Go语言中最常见的、也是常被人提及的设计模式 — 不要通过共享内存的方式进行通信，而是应该通过通信的方式共享内存。在很多主流的编程语言中，多个线程传递数据的方式一般都是共享内存，为了解决线程冲突的问题，我们需要限制同一时间能够读写这些变量的线程数量，这与 Go 语言鼓励的方式并不相同。

Thread1 ---> Memory ---< Thread2     // 多线程使用共享内存传递数据

虽然我们能在Go语言中也能使用共享内存加互斥锁进行通信，但是 Go 语言提供了一种不同的并发模型，也就是通信顺序进程（Communicating sequential processes，CSP）。
Goroutine和Channel分别对应CSP中的实体与传递消息的媒介，Go 语言中的 Goroutine会通过Channel传递数据。

Goroutine --> Channel --> Goroutine

上面的两个 Goroutine，一个会想 Channel中发送数据，另一个可以从Channel中接受数据，它们两者能够独立运行，并不存在直接关联，但是能通过Channel完成间接通信。

### FIFO（先入先出）
目前的 Channel收发操作均遵循了先入先出（FIFO）的设计，具体规则如下：

1. 先从 Channel 读取数据的 Goroutine会先接受到数据
2. 先向 Channel 发送数据的 Goroutine会先得到发送数据的权利

这种FIFO的设计是相对好理解的，但是 Go语言稍早版本的实现却不是严格遵守这一语义。
其中提出了有缓冲区的 Channel 在执行收发操作时没有遵循 FIFO 的规则。

先前的实现：
* 发送方会向缓冲区中写入数据，然后唤醒接收方，多个接收方会尝试从缓冲区中读取数据，如果没有读取到就会重新陷入休眠；
* 接收方会从缓冲区中读取数据，然后唤醒发送方，发送方会尝试向缓冲区写入数据，如果缓冲区已满就会重新陷入休眠；

这种基于重试的机制会导致 Channel 的处理不会遵循 FIFO 的原则。

### 无锁管道

锁是一种常见的并发控制技术，我们一般将锁分为乐观锁和悲观锁，即乐观并发控制和悲观并发控制，无锁（lock-free）队列更准确的描述是乐观并发控制的队列。乐观并发控制也叫乐观锁，但是它并不是真正的锁，很多人都会误以为乐观锁是一种真正的锁，然而它只是一种并发控制的思想。

乐观并发控制本质上是基于验证的协议，我们使用原子指令 CAS （compare-and-swap 或者
compare-and-set）在多线程同步数据，无锁队列的实现也依赖这一原子指令。

Channel 在运行时的内部表示是 runtime.hchan，该结构体中包含了一个用于保护成员变量的互斥锁，从某种程度上说，Channel 是一个用于同步和通信的有锁队列。使用互斥锁解决程序中可能存在的线程竞争问题是很常见的，我们能很容易地实现有锁队列。

然而锁导致的休眠和唤醒会带来额外的上下文切换，如果临界区6过小，加锁解锁导致的额外开销就会成为性能瓶颈。1994 年的论文 Implementing lock-free queues 就研究了如何使用无锁的数据结构实现先进先出队列7，而 Go 语言社区也在 2014 年提出了无锁 Channel 的实现方案，该方案将 Channel 分成了以下三种类型：

## 数据结构

Go 语言的 channel在运行时使用 runtime.hchan 结构体表示。我们在 Go 语言中创建新的Channel时， 实际上创建的都是如下所示的结构体：
```
type hchan struct {
    qcount uint
    dataqsiz uint
    buf unsafe.Pointer
    elemsize uint32
    closed uint32
    elemtyoe *_type
    sendx uint
    recvx uint
    recvq waitq
    sendq waitq

    lock mutex
}
```
runtime.hchan结构体中的五个字段 qcount、dataqsiz、buf、sendx、recvx 构建底层的循环队列：

* qcount：Channel 中元素的个数：
* dataqsiz：Channel 中循环队列的长度
* buf：Channel 的缓冲区数据的指针
* sendx：Channel 的发送操作处理到的位置
* recvx：Channel 的接受操作处理到的位置

除此之外，elemsize 和 elemtype 分别表示 当前 Channel 能够收发的元素大小和类型；sendq 和 存储了当前 Channel 由于缓冲区空间不足而阻塞的 Goroutine列表，这些等待队列使用双向链表 runtime.waitq 表示，链表中所有的元素都是 runtime.sudog 结构：
```
type waitq struct {
    first *sudog
    last *sudog
}
```

## 创建管道
Go 语言中所有的 Channel的创建都会使用 make 关键字。编译器会将 make(chan int, 10)
表达式被转换成 OMAKE 类型的节点，并在类型检查阶段将 OMAKE 类型的节点转换成 OMAKECHAN 类型：
```
func typecheck1(n *Node, top int) (res *Node) {
	switch n.Op {
	case OMAKE:
		...
		switch t.Etype {
		case TCHAN:
			l = nil
			if i < len(args) { // 带缓冲区的异步 Channel
				...
				n.Left = l
			} else { // 不带缓冲区的同步 Channel
				n.Left = nodintconst(0)
			}
			n.Op = OMAKECHAN
		}
	}
}
```
这一阶段会对传入 make 关键字的缓冲区大小进行检查，如果我们不向 make 传递表示缓冲区大小的参数，那么就会设置一个默认值 0，也就是当前的 Channel 不存在缓冲区。

OMAKECHAN 类型的节点最终都会在 SSA 中间代码生成阶段之前被转换成调用 runtime.makechan 或者 runtime.makechan64 的函数：
```
func walkexpr(n *Node, init *Nodes) *Node {
	switch n.Op {
	case OMAKECHAN:
		size := n.Left
		fnname := "makechan64"
		argtype := types.Types[TINT64]

		if size.Type.IsKind(TIDEAL) || maxintval[size.Type.Etype].Cmp(maxintval[TUINT]) <= 0 {
			fnname = "makechan"
			argtype = types.Types[TINT]
		}
		n = mkcall1(chanfn(fnname, 1, n.Type), n.Type, init, typename(n.Type), conv(size, argtype))
	}
}
```
runtime.makechan 和 runtime.makechan64 会根据传入的参数类型和缓冲区大小创建一个新的 Channel 结构，其中后者用于处理缓冲区大小大于 2 的 32 次方的情况，我们重点关注 runtime.makechan 函数：
```
func makechan(t *chantype, size int) *hchan {
	elem := t.elem
	mem, _ := math.MulUintptr(elem.size, uintptr(size))

	var c *hchan
	switch {
	case mem == 0:
		c = (*hchan)(mallocgc(hchanSize, nil, true))
		c.buf = c.raceaddr()
	case elem.kind&kindNoPointers != 0:
		c = (*hchan)(mallocgc(hchanSize+mem, nil, true))
		c.buf = add(unsafe.Pointer(c), hchanSize)
	default:
		c = new(hchan)
		c.buf = mallocgc(mem, elem, true)
	}
	c.elemsize = uint16(elem.size)
	c.elemtype = elem
	c.dataqsiz = uint(size)
	return c
}
```
上述代码根据 Channel 中收发元素的类型和缓冲区的大小初始化 runtime.hchan 结构体和缓冲区：

* 如果当前 Channel 中不存在缓冲区，那么就只会为 runtime.hchan 分配一段内存空间；
* 如果当前 Channel 中存储的类型不是指针类型，就会为当前的 Channel 和底层的数组分配一块连续的内存空间；
* 在默认情况下会单独为 runtime.hchan 和缓冲区分配内存；

在函数的最后会统一更新 runtime.hchan 的 elemsize、elemtype 和 dataqsiz 几个字段。

### 发送数据
当我们想要向 Channel发送数据时，就需要使用 ch <-i 语句，编译器会将他解析成 OSEND 节点并在 cmd/compile/internal/gc.walkexpr 函数中转换成 runtime.chansecd1：
```
func walkexpr(n *Node, init *Nodes) *Node {
	switch n.Op {
	case OSEND:
		n1 := n.Right
		n1 = assignconv(n1, n.Left.Type.Elem(), "chan send")
		n1 = walkexpr(n1, init)
		n1 = nod(OADDR, n1, nil)
		n = mkcall1(chanfn("chansend1", 2, n.Left.Type), nil, init, n.Left, n1)
	}
}
```
runtime.chansend1 只是调用了 runtime.chansend 并传入 Channel 和需要发送的数据。runtime.chansend 是向 Channel 中发送数据时最终会调用的函数，这个函数负责了发送数据的全部逻辑，如果我们在调用时将 block 参数设置成 true，那么就表示当前发送操作是一个阻塞操作：
```
func chansend(c *hchan, ep unsafe.Pointer, block bool, callerpc uintptr) bool {
	lock(&c.lock)

	if c.closed != 0 {
		unlock(&c.lock)
		panic(plainError("send on closed channel"))
	}
```

在发送数据的逻辑执行之前会先为当前 Channel 加锁，防止发生竞争条件。如果 Channel 已经关闭，那么向该 Channel 发送数据时就会报"send on closed channel" 错误并中止程序。

因为 runtime.chansend 函数的实现比较复杂，所以我们这里将该函数的执行过程分成以下的三个部分：
* 当存在等待的接收者时，通过 runtime.send 直接将数据发送给阻塞的接收者；
* 当缓冲区存在空余空间时，将发送的数据写入 Channel 的缓冲区；
* 当不存在缓冲区或者缓冲区已满时，等待其他 Goroutine 从 Channel 接收数据；

#### 直接发送
如果目标Channel没有被关闭，并且已经有处于读等待的Goroutine，那么 runtime.chansend 函数会从接受队列recvq中取出最先陷入等待的 Gorontine并直接向它发送数据：
```
if sg := c.recvq.dequeue(); sg != nil {
	send(c, sg, ep, func() { unlock(&c.lock) }, 3)
	return true
}
```

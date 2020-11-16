# 调度器
Go语言并发编程能力强劲，本身语言支持。

学习内容：
* 运行时调度器的实现原理
* 设计与实现原理
* 演变过程
* 运行时调度器相关的数据结构

谈到 Go语言调度器，我们绕不开的是操作系统、进程、线程这些概念，线程是操作系统调度时的最基本单元，而Linux在调度器并不区分进程和线程的调度，它们在不同操作系统上也有不同的实现，但是在大多数的实现中线程都属于进程：

多个线程可以同属于一个进程共享内存空间。因为多线程不需要创建新的虚拟内存空间，所以他们也不需要内存管理单元处理上下文的切换，线程之间的通信也是基于共享的内存进行的，与重量级的进程相比，线程显的比较轻量。

虽然线程比较轻量，但是在调度时也有比较大的额外开销。每个线程都会占用1M以上的内存空间，在对线程进行切换时不止会消耗较多的内存，恢复寄存器的内容还需要向操作系统申请或者销毁对应的资源，每一次线程上下文的切换都需要消耗 ~1us 左右的时间1，但是 Go 调度器对 Goroutine 的上下文切换约为 ~0.2us，减少了 80% 的额外开销。

Go语言的调度器通过使用与CPU数量相等的线程（虚拟核= 物理核心×2）减少线程频繁切换的内存开销，同时在每一个线程上执行额外开销更低的Goroutine来降低操作系统和硬件的负载。

## 设计原理
* 单线程调度器 0.x
* 多线程调度器 1.0
* 任务窃取调度器 1.1
* 抢占式调度器

### 任务窃取调度器

当前处理器本地的运行队列中不包含 Goroutine 时，调用 runtime.findrunnable 函数会触发工作窃取，从其它的处理器的队列中随机获取一些 Goroutine。

基于工作窃取的多线程调度器将每一个线程绑定到了独立的 CPU 上，这些线程会被不同处理器管理，不同的处理器通过工作窃取对任务进行再分配实现任务的平衡，也能提升调度器和 Go 语言程序的整体性能，今天所有的 Go 语言服务都受益于这一改动。

### 抢占式调度器
对 Go语言并发模型的修改提升了调度器的性能，但是1.1版本中的调度器仍然不支持抢占式调度，程序只能依靠 Goroutine主动出让CPU资源才能触发调度。Go 语言的调度器在 1.2 版本4中引入基于协作的抢占式调度解决下面的问题：

1. 某些Goroutine可以长长时间占用线程，造成其他Goroutine的饥饿
2. 垃圾回收需要暂停整个程序（Stop-the-world，STW），最长可能需要几分钟的时间6，导致整个程序无法工作

1.2 版本里的抢占式调度虽然能够缓解这个问题，但是抢占式调度是基于协作的，在之后很长一段时间里Go语言的调度器都有一些无法被抢占的边缘情况，例如：for 循环或者垃圾回收长时间占用线程，这些问题中的一部分直到 1.14 才被基于信号的抢占式调度解决。

#### 基于协作的抢占式调度

#### 基于信号的抢占式调度


## 数据结构
运行事调度器的三个重要组成部分— 线程 M、Goroutine G 和处理器 P：
1. G — 表示 Goroutine，它是一个待执行的任务
2. M — 表示操作系统的线程，它由操作系统的调度器调度与管理
3. P — 表示处理器，它可以被看做是运行在线程上的本地调度器

### G
Goroutine就是Go语言调度器中待执行的任务，它在运行时调度器中的地位与线程在操作系统中的差不多，但是它占用了更小的内存空间，也降低了上下文切换的开销。

Goroutine只存在于Go语言的运行时，它是 Go语言在用户态提供的线程。作为一中粒度更细的资源调度单元，如果使用得当能够在高并发的场景下更高效的利用机器的CPU

Goroutine 在go语言运行时使用私有的结构体 runtime.g 表示。这个私有的结构体非常复杂，总共包括 40 多个用于表示各种状态的成员变量。我们只学习其中一部分，首先介绍与栈相关的两个字段：
```
type g struct {
    stack stack         // 描述了实际的栈内存
    
    // stackguard0 是对比 Go 栈增长的 prologue 的栈指针
    // 如果 sp 寄存器比 stackguard0 小（由于栈往低地址方向增长），会触发栈拷贝和调度
    // 通常情况下：stackguard0 = stack.lo + StackGuard，但被抢占时会变为 StackPreempt
    stackguard0 uintptr  // uintptr，一个足够大的无符号整型， 用来表示任意地址，可以进行数值计算。
}


// stack 描述了 goroutine 的执行栈， 栈区间为[lo, hi)，在栈两边没有任何隐式的数据结构。
type stack struct {   
    lo uintptr
    hi uintptr
}
```

其中 stack 字段描述了当前Goroutine的栈内存的范围 [stack.lo, stack.hi]，另一个字段 stackguard0 可以用于调度器抢占式调度。除了 stackgurad0 之外， Goroutine中还包含另外三个与抢占密切相关的字段：
```
type g struct {
    preempt bool        // 抢占信号
    preemptStop bool    // 抢占时将信号修改成 `_Gpreempted`
    preemptShrink bool  // 在同步安全点收缩栈
}
```

Goroutine 与我们在前面章节提到的 defer和 panic也有千丝万缕的联系， 每个 Goroutine 上都持有两个分别存储 defer 和 panic 对应结构体的链表：
```
type g struct {
    _panic *_panic         // 最内测的 panic 结构体
    _defer *_defer         // 最内测的u延迟函数结构体
}
```

其他重要或者有趣的字段：
```
type g struct {
    m *m
    scded gobuf
    atomicstatus uint32
    goid int64
}
```
* m — 当前 Goroutine 占用的线程，可能为空
* atomicstatus — Goroutine 的状态
* sched — 存储 Goroutine 的调度相关的数据
* goid — Goroutine 的 ID，该字段对开发者不可见，Go 团队认为引入 ID 会让部分 Goroutine 变得更特殊，从而限制语言的并发能力

需要展开学习 sched 字段的 runtime.gobuf 结构体中包含那些内容：
```
type gobuf struct {
    sp  uintptr
    pc  uintptr
    g   guintptr
    ret sys.Uintreg
    ...
}
```

* sp — 栈指针（Stack Pointer）
* pc — 程序计数器（Program Counter）
* g — 持有 runtime.gobuf 的 Goroutine
* ret — 系统调用的返回值

这些内容会在调度器保存或者恢复上下文的时候用到，其中的栈指针和程序计数器会用来存储或者恢复寄存器中的值，改变程序即将执行的代码。

结构体 runtime.g 的 atomicstatus 字段就存储了当前 Goroutine的 状态。除了几个已经不被使用的以及与 GC 相关的状态之外，Goroutine 可能处于以下 9 个状态：
状态|描述|
-|-|
_Gidle|刚刚被分配并且还没有被初始化
_Grunnable|没有执行代码，没有栈的所有权，存储在运行队列中
_Grunning|可以执行代码，拥有栈的所有权，被赋予了内核线程 M 和处理器 P
_Gsyscall|正在执行系统调用，拥有栈的所有权，没有执行用户代码，被赋予了内核线程 M 但是不在运行队列上
_Gwaiting|由于运行时而被阻塞，没有执行用户代码并且不在运行队列上，但是可能存在于 Channel 的等待队列上
_Gdead|没有被使用，没有执行代码，可能有分配的栈
_Gcopystack|栈正在被拷贝，没有执行代码，不在运行队列上
_Gpreempted|由于抢占而被阻塞，没有执行用户代码并且不在运行队列上，等待唤醒
_Gscan|GC 正在扫描栈空间，没有执行代码，可以与其他状态同时存在

上述状态中比较常见的是 _Grunnable、_Grunning、_Gsyscall、_Gwaiting 和 _Gpreempted，然后重点学习这个几个状态。Goroutine 的状态迁移是一个复杂的过程，触发 Goroutine 状态迁移的方法也很多，也是学习一部分迁移路线。

虽然Goroutine在运行时中定义的状态非常多且复杂，但是我们可以将这些不同的状态聚合成最终的三种形态：等待中、可运行、运行中，在运行期间我们可以在这种三种状态来回切换：

* 等待中：Goroutine 正在等待某些条件满足，例如：系统调用结束等，包括 _Gwaiting、_Gsyscall 和 _Gpreempted 几个状态；
* 可运行：Goroutine 已经准备就绪，可以在线程运行，如果当前程序中有非常多的 Goroutine，每个 Goroutine 就可能会等待更多的时间，即 _Grunnable；
* 运行中：Goroutine 正在某个线程上运行，即 _Grunning；

### M
Go 语言并发模型中的M是操作系统线程。调度器最可以创建10000个线程，但是其中大多数的线程都不会执行用户代码（可能陷入系统调用），最多只会有 GOMAXPROCS 个活跃线程正常运行。

在默认情况下，运行时会将 GOMAXPROCS 设置成设置成当前机器的核数（逻辑核或者虚拟核），当然我们也可以使用 runtime.GOMAXPROCS 来改变程序中最大的线程数。

在默认情况下，一个四核的机器上会创建四个活跃的操作系统线程，每一个线程都对应一个运行时中的runtime.m结构体。

在大多数情况下，我们都会使用Go的默认设置，也就是线程等于CPU个数，在这种情况下不会触发操作系统的线程调度和上下文切换，所有的调度都会发生在用户态，由 Go 语言调度器触发，能够减少非常多的额外开销。

操作系统线程在 Go语言中会使用私有结构体 runtime.m 来表示。这个结构体也是包含了十几个私有的字段，先学习与 Goroutine直接相关的字段：
```
type m struct {
    g0 *g
    curg *g
    ...
}
```

其中 g0 是持有调度栈的 Goroutine，curg 是当前线程上运行的用户 Goroutine， 这也是操作系统线程唯一关系的 Goroutine。

g0 是一个运行时比较特殊的 Goroutine，它会深度参与运行时的调度过程，包括 Goroutine的创建、大内存分配和 CGO函数的执行。runtime.m 结构体中还存在着三个处理器字段，它们分别代表正在运行代码的处理器P，暂存的处理器 nextp 和执行系统调用之前的使用线程的处理器 oldp：

```
type m struct {
    p   puintptr
    nextp   puintptr
    oldp    puintptr
}
```
除此之外，runtime.m中还存在着大量与线程状态、锁、系统调用有关的字段。

### P
调度器中的处理器 P 是线程和 goroutine 的中间层，它能提供线程需要的上下文环境，也会负责调度线程上的等待队列，通过处理器P的调度，每一个内核线程都能够执行多个Goroutine，它能在Goroutine进行i/o操作时及时切换，提高线程的利用率。

因为调度器在启动时就会创建 GOMAXPROCS 个处理器，所以 Go 语言程序的处理器数量一定会等于 GOMAXPROCS，这些处理器会绑定到不同的内核线程上并利用线程的计算资源运行 Goroutine。

runtime.p 是处理器的运行时表示，作为调度器的内部实现，它包含的字段也非常多，其中包括与性能追踪、垃圾回收和计时器相关的字段。现在，我们主要关注处理器中的线程和运行队列：
```
type p struct {
    m       muintptr

    runqhead    uint32
    runqtail    uint32
    runq        [256]guintptr
    runnext     guintptr
}
```
反向存储的线程维护着线程与处理器之间的关系，而 runhead、runqtail和runq三个表示处理器持有的所有的运行队列，其中存储着待执行的Goroutine列表，而runnext中是线程下一个要执行的Goroutine.
runtime.p 结构体中的状态 status字段会是以下五种中的一种：

状态|描述|
-|-|
_Pidle|处理器没有运行用户代码或者调度器被空闲队列或者改变其状态的结构持有，运行队列为空|
_Prunning|被线程M持有，并且正在执行用户代码或者调度器|
_Psyscall|没有执行用户代码，当前线程陷入系统调用|
_Pgcstop|被线程M持有，当前处理器由于由于垃圾回收被暂停|
_Pdead|当前处理器已经不被使用了|

通过分析处理器 P 的状态，我们能够对处理器的工作过程有一些简单理解，例如处理器在执行用户代码时会处于 _Prunning 状态，在当前线程执行 I/O 操作时会陷入 _Psyscall 状态。

## 调度器启动
了解学习调度器的启动过程对理解调度器的实现原理很有帮助，运行时通过 runtime.schedinit 函数初始化调度器：
```
func schedinit() {
    _g_ := getg()
    ...

    sched.maxmcount = 10000
    ...

    sched.lastpoll = uint64(nanotime())
    procs := ncpu
    if n, ok := atoi32(gogetenv("GOMAXPROCS")); ok && n > 0 {
        procs = n
    }

    if procresize(procs) != nil {
        throw("unknown runnable goroutine during bootstrap")
    }
}
```
在调度器初始化程序执行过程中会将 maxmcount 设置成10000，这也就是一个Go语言程序能够创建的最大的线程数，虽然最多可以创建10000个线程，但是同时运行的线程还是由 GOMAXPROCS 变量控制。

我们从环境变量 GOMAXPROCS 获取了程序能够运行的最大处理器之后就会调用 runtime.procresize 更新程序中处理器的数量，在这时整个程序不会执行任何 Goroutine，调度器也会进入锁定状态，runtime.procresize 的执行过程如下：

1. 如果全局变量 allp (所有的处理器) 切片中的处理器数量少于期望数量，就会对切片进行扩容
2. 使用 new 创建新的处理器结构体并调用 runtime.p.init 方法初始化刚扩容的处理器
3. 通过指针将 m0 和处理器 allp[0] 绑定到一起
4. 调用 runtime.p.destory 方法释放不再使用的处理器结构
5. 通过截断改变全局变量 allp 的长度保证与期望处理器数量相等
6. 将除 allp[0] 之外的处理器 P 全部摄制成 _Pidle 并加入到全局的空闲队列中

调用 runtime.procresize 就是调度器启动的最后一步，在这一步过后调度器会完成相应数量处理器的启动，等待用户创建运行新的 Goroutine 并为 Goroutine 调度处理器资源。


## 创建 Goroutine
想要启动一个新的 Goroutine 来执行任务，需要用 Go语言中的 Go关键字，这个关键字会在编译期间通过以下方法 cmd/compile/internal/gc.state.stmt 和 cmd/compile/internal/gc.state.call 两个关键字转化成runtime.newproc 函数调用：
```
func (s *state) call(n *Node, k callKind) *ssa.Value {
    if k == callDeferStack {
    ...
    } else {
        switch {
            case k == callGo:
                call = s.newValue(ssa.OpStaticCall, types.TypeMem, newproc, s.mem())
            default:
        }
    }
    ...
}
```

编译器会将所有的 go 关键字转换成 runtime.newproc 函数，该函数会接受大小和表示函数的指针 funcval。在这个函数中我们还会获取 goroutine 以及调用放的程序计时器，然后调用 runtime.newproc1 函数：
```
func newproc(size int32, fn *funcval) {
    argp := add(unsafe.Pointer(&fn), sys.PtrSize)
    gp := getg()

    pc := getcallerpc()
    newproc1(fn, (*uint8)(argp), size, gp, pc)
}
```

runtime.newproc1 会根据传入参数初始化一个 g 结构体，我们可以将该函数分成以下几个部分介绍它的实现：

1. 获取或者创建新的 Goroutine 结构体
2. 将传入的参数移到 Goroutine 的栈上
3. 更新 Goroutine调度相关的参数
4. 将Goroutine加入处理器的运行队列

```
func newproc1(fn *funcval, argp *uint8, narg int32, callerpg *g, callerpc uintptr) {
    _g_ := getg()
    siz := narg
    siz := (siz + 7) &^ 7

    _p_ := _g_.m.p.ptr()
    newg := gfget(_g_)
    if newg == nil {
        newg = malg(_StackMin)
        casgstatus(newg, _Gidle, _Gdead)
        allgadd(newg)
    }
}
```
上述代码会先从处理器的 gFree 列表中查找空闲的 Goroutine，如果不存在空闲的Goroutine，就会通过 runtime.malg 函数创建一个栈大小足够的新结构体。

接下来，我们会调用 runtime.memmove 函数将 fn 函数的全部参数拷贝到栈上， argp和narg 分别是参数的内存空间和大小，我们在该方法中会直接将所有的参数对应的内存空间整片的拷贝到栈上：
```
...
	totalSize := 4*sys.RegSize + uintptr(siz) + sys.MinFrameSize
	totalSize += -totalSize & (sys.SpAlign - 1)
	sp := newg.stack.hi - totalSize
	spArg := sp
	if narg > 0 {
		memmove(unsafe.Pointer(spArg), unsafe.Pointer(argp), uintptr(narg))
	}
...
```

拷贝了栈上的参数之后，runtime.newproc1 会设置新的 Goroutine结构体参数，包括栈指针、程序计数器并更新其到_Grunnable：

```
...
	memclrNoHeapPointers(unsafe.Pointer(&newg.sched), unsafe.Sizeof(newg.sched))
	newg.sched.sp = sp
	newg.stktopsp = sp
	newg.sched.pc = funcPC(goexit) + sys.PCQuantum
	newg.sched.g = guintptr(unsafe.Pointer(newg))
	gostartcallfn(&newg.sched, fn)
	newg.gopc = callerpc
	newg.startpc = fn.fn
	casgstatus(newg, _Gdead, _Grunnable)
	newg.goid = int64(_p_.goidcache)
	_p_.goidcache++
```

在最后，该函数会将初始化好的 Goroutine 加入到处理器的运行队列并在满足条件是调用 runtime.wakep 函数唤醒新的处理执行 Goroutine；
```
    ...
	runqput(_p_, newg, true)

	if atomic.Load(&sched.npidle) != 0 && atomic.Load(&sched.nmspinning) == 0 && mainStarted {
		wakep()
	}
}
```

我们在分析 runtime.newproc1 函数的过程中，省略了两个比较重要的过程，分别是用于获取结构体的 runtime.gfget、runtime.malg 函数、将 Goroutine 加入运行队列的 runtime.runqput 以及调度信息的设置过程。

### 初始化结构体
runtime.gfget 通过两种不同的方式获取新的 runtime.g结构体：

1. 从 Goroutine 所在的 gFree 列表或者调度器的 sched.gFree 列表中获取 runtime.g结构体
2. 调用 runtime.malg 函数生成一个新的 runtime.g 函数并将并将当前结构体的追加到全局的 Goroutine列表 allgs 中

runtime.gfget 中包含两部分逻辑，它会根据 gFree列表中 Goroutine 数量作出不同的决策：

1. 当处理器的 Goroutine列表为空时，会将调度器持有的空闲 Goroutine 转移到当前处理器上，直到 gFree 列表中的 Goroutine数量达到32；
2. 当处理器的 Goroutine数量充足时，会从列表头部返回一个新的 Goroutine

```
func gfget(_p_ *p) *g {
retry:
    if _p_.gFree.empty() && (!sched.gFree.stack.empty() || !sched.gFree.noStack.empty()) {
        for _p_.gFree.n < 32 {
            gp := sched.gFree.stack.pop()
            if gp == nil {
                gp = sched.gFree.noStack.pop()
                if gp == nil {
                    break
                }
            }
            _p_.gFree.push(gp)
        }
        goto retry
    }

    gp := _p_.gFree.pop()
    if gp == nil {
        return nil
    }

    return gp
}
```

当调度器 gFree 和 处理器的 gFree 列表都不存在结构体时，运行时会调用 runtime.malg 初始化一个新的 runtime.g 结构体， 如果申请的堆栈大小大于0，在这里我们会通过 runtime.stackalloc分配2kb的栈空间：

```
func malg(stacksize int32) *g {
    newg := new(g)
    if stacksize >= 0 {
        stacksize = round2(_StackSystem + stacksize)
        newg.stack = stackalloc(uint32(stacksize))

        newg.stackguard0 = newg.stack.lo + _StackGuard
        newg.stackguard1 = ^uintptr(0)
    }

    return newg
}
```
runtime.malg 返回的 Goroutine 会存储到全局变量 allgs 中。

简单总结一下， runtime.newproc1 会从处理器或者调度器的缓存中获取新的结构体，也可以调用 runtime.malg 创建新的结构体。

### 运行队列
runtime.runqput 函数会将新创建的 Gorotine 加入运行队列上，这既可能是全局的运行队列，也可能是处理器本地的运行队列：

```
func runqput(_p_ *p, gp *g, next bool) {
    if next {
        retryNext:
            oldnext := _p_.runnext
            if !_p_.runnext.cas(oldnext, guintptr(unsafe.Pointer(gp))) {
                goto retryNext
            }
            if oldnext == 0 {
                return
            }
            gp = oldnext.ptr()
    }

retry:
    h := atomic.LoadAcq(&_p_.runqhead)
    t := _p_.runqtail
    if t-h < uint32(len(_p_.runq)) {
        _p_.runq[t%uint32(len(_p_.runq))].set(gp)
        atomic.StoreRel(&_p_.runqtail, t+1)
        return
    }

    if runqputslow(_p_, gp, h, t) {
        return
    }

    goto retry
}
```

1. 当 next 为 true时，将 Goroutine 设置到处理器的 runnext 上作为下一个处理器执行的任务；
2. 当 next 为 false并且本地运行的队列还有剩余的空间时，将 Goroutine 加入到处理器持有的本地运行队列
3. 当处理器的本地运行队列已经没有剩余空间时就会把本地队列中的一部分 Goroutine 和待加入的 Goroutine 通过 runqputslow 添加到调度器持有的全局运行队列上

处理器本地的运行队列是一个使用数组构成的环形链表，它最多可以存储256个待执行任务。

小结一下：Go 语言中有两个运行队列，其中一个是处理器本地的运行队列，另外一个是调度器持有的全局运行队列，只有在本地运行队列没有剩余空间时才会使用全局队列。

### 调度信息
运行时创建 Goroutine时会通过以下代码设置调度的相关信息，前两行代码分别会将程序计数器和Goroutine设置成 runtime.goexit 函数和新创建的 Goroutine：

```
    ...
    newg.sched.pc = funcPC(goexit) + sys.PCQuantum
    newg.sched.g = guintptr(unsafe.Pointer(newg))
    gostartcallfn(&newg.sched, fn)
    ...
```

但是这里的调度信息 sched 不是初始化后的 Goroutine 的最终结果，经过 runtime.gastartcallfn 和 runtime.gostartcall 两个函数的处理：

```
func gostartcallfn(gobuf *gobuf, fv *funcval) {
    gostartcall(gobuf, unsafe.Pointer(fv.fn), unsafe.Pointer(fv))
}

func gostartcall(buf *gobuf, fn, ctxt unsafe.Pointer) {
    sp := buf.sp
    if sys.RegSize > sys.PtrSize {
        sp -= sys.PtrSize
        *(*uintptr)(unsafe.Pointer(sp)) = 0
    }

    sp -= sys.PtrSize
    *(*uintptr)(unsafe.Pointer)(sp) = buf.pc
    buf.sp = sp
    buf.pc = uintptr(fn)
    buf.ctxt = ctxt
}
```
调度信息的 sp 存储了 runtime.goexit 函数的计数器，而 pc 中存储了传入函数的程序计数器。因为 pc 寄存器的作用就是存储程序接下来运行的位置，所以这里的 pc 的使用较好理解。但是 sp 中存储的 runtime.goexit 就会让人感到困惑，我们需要配合下面的调度循环来理解 sp 的作用。

## 调度循环
调度器启动之后， Go 语言运行时会调用 runtime.mstart 以及 runtime.mstart1，前者会初始化 g0 的 stackguard0 和 stackguard1字段，后者会初始化线程并调用 runtime.schedule 进入调度循环：
```
func schedule() {
    _g_ := getg()

top:
    var gp *g
    var inheritTime bool

    if gp == nil {
        // // 该p上每进行61次就从全局队列中获取一个g
        if _g_.m.p.ptr().schedtick%61 == 0 && sched.runqsize > 0 {
            lock(&sched.lock)
            gp = globrunqget(_g_.m.p.ptr(), 1)
            unlock(&sched.lock)
        }
    }

    if gp == nil {
        gp, inheritTime = runqget(_g_.m.p.ptr())
    }

    if gp == nil {
        gp, inheritTime = findrunnable()
    }
    execute(gp, inheritTime)
}
```

runtime.schedule 函数会从不同的地方查找待执行的 Goroutine

1. 为了保证公平，当全局运行队列有待执行的 Goroutine时，通过 schedtick 保证有一定的几率会从全局的运行队列中查找对应的 Goroutine；

2. 从处理器本地的运行队列中查找待执行的 Goroutine

3. 如果前两种方法都没找到 Goroutine，就会通过runtime.findrunnable进行阻塞的查找Goroutine

runtime.findrunnable 函数的实现非常复杂，上述执行过程是经过大量简化的，总而言之，当前函数一定会返回一个可执行的 Goroutine，如果当前不存在就会阻塞等待。

接下由 runtime.execute 函数执行获取的 Goroutine，做好准备工作之后，它会通过 runtime.gogo将 Goroutine调度到当前线程上。

```
func execute(gp *g, inheritTIme bool) {
    _g_ := getg()
    _g__.m.curg = gp
    gp.m = _g_.m
    
    casgstatus(gp, _Grunnable, _Grunning)
    gp.waitsince(0)
    gp.preempt = false

    gp.stackguard0 = gp.stack.lo + _StackGuard

    if !inheritTime {
        _g_.m.p.ptr().schedtick++
    }

    gogo(&gp.sched)
}
```
runtime.gogo（汇编） 在不同处理器架构上的实现都不同，但是不同的实现也都是大同小异，下面是该函数在 386 架构上的实现：

```
TEXT runtime·gogo(SB), NOSPLIT, $8-4
	MOVL buf+0(FP), BX     // 获取调度信息
	MOVL gobuf_g(BX), DX
	MOVL 0(DX), CX         // 保证 Goroutine 不为空
	get_tls(CX)
	MOVL DX, g(CX)
	MOVL gobuf_sp(BX), SP  // 将 runtime.goexit 函数的 PC 恢复到 SP 中
	MOVL gobuf_ret(BX), AX
	MOVL gobuf_ctxt(BX), DX
	MOVL $0, gobuf_sp(BX)
	MOVL $0, gobuf_ret(BX)
	MOVL $0, gobuf_ctxt(BX)
	MOVL gobuf_pc(BX), BX  // 获取待执行函数的程序计数器
	JMP  BX 
```
该 runtime.gogo 函数设计的非常巧妙，它从 runtime.gobuf 中取出了 runtime.goexit 的程序的程序计数器和待执行函数的程序计数器，其中：

1. runtime.goexit的程序计数器被放到了栈 SP 上；
2. 待执行函数的程序计数器被放到了寄存器 BX 上；

在函数调用的一节中，我们学习过 Go语言的调用惯例，正常的函数调用都会使用 CALL 指令，该指令会将调用方的返回地址加入栈寄存器 SP 中，然后跳转到目标函数；当目标函数返回后，会从栈中查找调用的地址并跳转回调用方继续执行剩下的代码。

runtime.gogo 就利用了 Go语言的调用惯例成功模拟这一调用过程，通过几个关键指令模拟 CALL 的过程：

```
MOVL gobuf_sp(BX), SP
MOVL gobuf_pc(BX), BX
```

[图片详情](https://img.draveness.me/2020-02-05-15808864354661-golang-gogo-stack.png) 

调用 JMP 指令后的（runtime.gogo）栈中数据，如上链接所示。当 Goroutine 中的运行函数返回时就会跳转到 runtime.goexit 所在位置执行该函数：

```
TEXT runtime·goexit(SB).NOSPLIT, $0-0
    CALL    runtime·goexit(SB)

func goexit1() {
    mcall(goexit0)
}
```

经过一系列函数调用之后，我们最终在当前线程的 g0 的栈上调用 runtime.goexit 函数。该函数会将当前刚 执行过的 Goroutine 置空（转换为 _dead 状态、清理其中的字段、移除 Goroutine和线程的关联) 并调用 runtime.gfput 重新加入处理器的 Goroutine 空闲列表 gFree：

```
func goexit0(gp *g) {
    _g_ := getg()
    casgstatus(gp, _Grunning, _Gdead)
    gp.m = nil
    ...

    gp.param = nil
    gp.labels = nil
    gp.timer = nil

    dropg()

    gfput(_g_.m.p.ptr(), gp)
    schedule()
}
```
在最后 runtime.goexit0 函数会重新调用 runtime.schedule 触发新的 Goroutine 调度，我们可以认为调度循环永远都不会返回。

## 触发调度
调度器的 runtime.schedule 函数重新选择 Goroutine 在线程上执行，所以我们只要找到该函数的调用方就能找到所有出发调度的时间点。经过一番查找，发现了该大佬（draveness）分析和整理的树形结构：

[调度时间点](https://img.draveness.me/2020-02-05-15808864354679-schedule-points.png)

除了上图可能触发调度的时间点，运行时还会在线程启动 runtime.mstart 和 Goroutine执行结束 runtime.goexit0 触发调度。我们在重点学习运行时触发调度的几个路径：

* 主动挂起：runtime.gopark -> runtime.park_m
* 系统调用：runtime.exitsyscall -> runtime.exitsyscall0
* 协作式调度：runtime.Gosched -> runtime.goschd_m -> runtime.goschedImpl
* 系统监控：runtime.sysmon -> runtime.retaka -> runtime.preemptone

我们在这里说的（介绍）的调度时间点不是直接将线程的运行权交给其他任务，而是通过调度器的 runtime.schedule 重新调度。

### 主动挂起
runtime.gopark 是触发调度最常见的方法，该函数会将当前 Goroutine 暂停，被暂停的任务不会放回队列，我们来学习下该函数的实现原理：

```
func gopark(unlockf func(*g, unsafe.Pointer) bool, lock unsafe.Pointer, reason waitReason, traceEv byte, traceskip int) {
    if reason != waitReasonSleep {
        checkTimeouts()
    }

    mp := acquirem()
    gp := mp.curg

    status := readgstatus(gp)
    if status != _Grunning && status != _Gscanrunning {
        throw("gopark: bad g status")
    }

    mp.waitlock = lock
    mp.waitunlockf = unlockf
    gp.waitreason = reason
    mp.waittraceev = traceEv

    mp.waittraceskip = traceskip

    releasem(mp)
    // mcall(park_m)
}
```
该函数会通过 runtime.mcall 在切换到 g0 的栈上调用 runtime.park_m 函数：
```
func park_m(gp *g) {
    _g_ := getg()

    casgstatus(gp, _Grunning, _Gwaiting)
    dropg()

    schedule()
}
```

该函数会将当前 Goroutine的状态从_Grunning 切换值 _Gwaiting, 调用 runtime.dropg 移除线程和 Goroutine之间的关联， 在这之后就可以调用 runtime.schedule触发新一轮的调度了。

当 Goroutine等待的特定条件满足后，运行时会调用 runtime.goready 将 因为调用 runtime.gopark 而陷入休眠的 Goroutine 唤醒。

```
func goready(gp *g, traceskip int) {
	systemstack(func() {
		ready(gp, traceskip, true)
	})
}

func ready(gp *g, traceskip int, next bool) {
	_g_ := getg()

	casgstatus(gp, _Gwaiting, _Grunnable)
	runqput(_g_.m.p.ptr(), gp, next)
	if atomic.Load(&sched.npidle) != 0 && atomic.Load(&sched.nmspinning) == 0 {
		wakep()
	}
}
```

runtime.ready 会将准备就绪的 Goroutine 的状态切换在至 _Grunnable 并将其加入处理器的运行队列中，等待调度器的调度。

### 系统调用

系统调用到底是什么？ 

（系统调用是操作系统内核提供给用户空间程序的一套标准接口。通过这套接口，用户态程序可以受限地访问硬件设备，从而实现申请系统资源，读写设备，创建新进程等操作。）

其实，系统调用的函数是操作系统提供的，也就是如果我们想用系统的功能，你就必须使用系统调用。比如上面讲的文件操作（创建，读取，更新，删除），网络操作（监听端口，接受请求和数据，发送数据），最近很火的docker，实现方式也是系统调用（NameSpace + CGroup等），简单的在命令行执行 echo hello，也用到了系统调用。


系统调用也会触发运行时调度器的调度，为了处理特殊的系统调用，甚至在 Goroutine 中加入了 _Gsyscall 状态， Go 语言通过 syscall.Syscall 和 syscall.RawSyscall 等使用使用汇编语言编写的方法封装了操作系统提供的所有的系统调用，其中 syscall.Syscall 的实现如下：
```
#define INVOKE_SYSCALL	INT	$0x80

TEXT ·Syscall(SB),NOSPLIT,$0-28
	CALL	runtime·entersyscall(SB)
	...
	INVOKE_SYSCALL
	...
	CALL	runtime·exitsyscall(SB)
	RET
ok:
	...
	CALL	runtime·exitsyscall(SB)
	RE在通过汇编指令 INVOKE_SYSCALL 执行系统调用前后，上述函数会调用运行时的 runtime.entersyscall 和 runtime.exitsyscall，正是这一层包装能够让我们在陷入系统调用前触发运行时的准备和清理工作。
```

[Go语言系统调用图片](https://img.draveness.me/2020-02-05-15808864354688-golang-syscall-and-rawsyscall.png)

不过出于性能的考虑，如果这次系统调用不需要运行时参与，就会使用 syscall.RawSyscall 简化这一过程，不再调用运行时函数。
这里包含Go语言对Linux386架构上不同的系统调用分类，我们按需决定是否需要运行时的参与：

系统调用|类型|
-|-|
SYS_TIME|RawSyscall
SYS_GETTIMEOFDAY|RawSyscall
SYS_SETRLIMIT|RawSyscall
SYS_GETRLIMITF|RawSyscall
SYS_EPOLL_WAIT|Syscall
...|...|

由于直接进行系统调用会阻塞当前的线程，所以只有立刻返回的系统调用才可能会被设置成 RawSyscall 类型，例如：SYS_EPOLL_CREATE、SYS_EPOLL_WAIT（超时时间为0）、SYS_TIME等。

Syscall在开始和结束的时候，会分别调用runtime中的进入系统和退出系统的函数，所以Syscall是受调度器控制的，因为调度器有开始和结束的的事件。而RawSyscall则不受调度器控制，RawSyscall 可能会导致其他正在运行的线程（协程）阻塞，调度器可能会在一段时间后运行它们，但是也有可能不会。所以，我们在进行系统调用的时候，应该极力避免使用RawSyscall，除非你确定这个操作是非阻塞的。

正常的系统调用过程相比之下比较复杂，接下来我们将分别学习进入系统调用前的准备工作和系统调用结束后的收尾工作。

#### 准备工作
runtime.entersyscall 函数会在获取当前程序计时器和栈位置之后调用
runtime.reentersyscall，它会完成 Goroutine进入系统调用前的准备工作。

```
func reentersyscall(pc, sp uintptr) {
    _g_ := getg()
    _g_.m.locks++
    _g_stackguard0 = stackPreempt   // 1. 禁止线程上发生的抢占，防止出现内存不一致的问题
    _g_.throwsplit = true // 2. 保证当前函数不会触发栈分裂或者增长

    save(pc, sp)    // 3. 保存当前的程序计数器 PC 和栈指针 SP 中的内容
    _g_.syscallsp = sp
    _g_.syscallpc = pc
    casgstatus(_g_, _Grunning, _Gsyscall) // 4. 将Goroutine的状态更新至 _Gsyscall

    _g_.m.syscalltick = _g_m.p.ptr().syscalltick
    _g_.m.mcache = nil
    pp := _g_.m.p.ptr()
    pp.m = 0
    _g_.m.oldp.set(pp)
    _g_.m.p = 0


    // 5. 将 Goroutine 的处理器和线程暂时分离并更新处理器的状态到 _Psyscall
    atomic.Store(&pp.status, _Psyscall)
    if sched.gcwaiting != 0 {
        systemstack(entersyscall_gcwait)
        save(pc, sp)
    }

    _g_m.lockso--   // 6. 释放当前线程上的锁
}
```

需要注意的是 runtime.reentersyscall 方法会使处理器和线程的分离，当前线程会陷入系统调用，等待返回，当前的线程上的锁被释放后，会有其他的 Goroutine抢占处理器的资源

#### 恢复工作

当系统调用结束后，会调用退出系统调用的函数 runtime.exitsyscall 为当前 Goroutine 重新分配资源，该函数有两个不同的执行路径：

1. 调用 exitsyscallfast 函数
2. 切换至调度器的 Goroutine 并调用 exitsyscall0函数

```
func exitsyscall() {
    _g_ := getg()
    oldp := _g_.m.oldp.ptr()
    _g_.m.oldp = 0
    if exitsyscallfast(oldp) {
        _g_.m.p.ptr().syscalltick++
        casgstatus(_g_, _Gsyscall, _Grunning)
        ...


        return
    }

    mcall(exitsyscall0)
    _g_.m.p.ptr().syscalltick++
    _g_.throwsplit = false

}
```

这两种不同的路径分别会通过不同的方法查找一个用于执行当前 Goroutine 的处理器P，快速路径 exitsyscallfast 中包含两个不同的分支：

1. 如果 Goroutine 的原处理器处于 _Psyscall 状态，就会直接调用 wirep 将 Goroutine 与处理器进行关联；
2. 如果调度器中存在闲置的处理器，就会调用 acquirep 函数使用闲置的处理器处理当前 Goroutine；

另外一个相对较慢的路径 exitsyscall0 就会将当前的 Goroutine 切换至 _Grunable 状态，并移除线程M 和 Goroutine 的关联：

1. 当我们通过 pidleget 获取到闲置的处理器时就会在该处理器上执行 Goroutine
2. 在其它情况下，我们会将当前 Goroutine 放到全局的运行队列中，等待调度器的调度

无论哪种情况，我们在这个函数中都会调用 schedule 函数触发调度器的调度

### 协作式调度

之前就学习过 go 语言基于协作式和型号抢占两种抢占式调度，在这接着学习 Go语言的协作式调度。runtime.Gosched 就是主动出让处理器，允许其他 Goroutine 运行。该函数无法自动挂起 Goroutine，调度器会自动调度当前的 Goroutine：
```
func Gosched() {
    checkTimeouts()
    mcall(gosched_m)
}

func gosched_m(gp *g) {
    goschedImpl(gp)
}

func goschedImpl(gp *g) {
    casgstatus(gp, _Grunning, _Grunable)
    dropg()
    lock(&sched.lock)
    globrunqput(gp)
    unlock(&sched.lock)

    schedule()
}
```

兜兜转转，最终在 g0 的栈上调用 runtime.goschedImpl函数，运行时会更新 Goroutine的状态到 _Grunnable，让出当前的处理器并将 Goroutine 重新放回全局队列，该函数在最后会重新调用 runtime.schedule 重新触发调度。

## 线程管理

Go 语言的运行时会通过调度器改变线程的所有权，它也提供了 runtime.LockOSThread 和 runtime.UnlockOSThread 让我们有能力绑定 Goroutine和线程完成一些比较特殊的操作，Goroutine 应该在调用操作系统服务或者依赖线程状态的非 Go语言库时调用 runtime.LockOSThread 函数，例如：C 语言的图形库等。

runtime.LockOSThread 会通过如下所示的代码绑定 Goroutine和当前线程：

```
func LockOSThread() {
    if atomic.Load(&newHandoff.havaTemplateThread) == 0 && GOOS != "plan9" {
        startTemplateThread()
    }

    _g_ := getg()
    _g_.m.lockedExt++
    dolockOSThread()
}

func dolockOSThread() {
    _g_ := getg()
    _g_.mlockedg.set(_g_)
    _g_.lockedm.set(_g_.m)
}
```

runtime.dolockOSThread 会分别设置线程的 lockedg字段和 Goroutine 的 lockedm 字段，这两行代码会绑定线程和 Goroutine。

当 Goroutine 完成了特定的操作之后，就会调用以下函数 runtime.UnlockOSThread 分离 Goroutine 和线程：

```
func UnlockOSThread() {
    _g_ := getg()
    if _g_.m.lockedExt == 0 {
        return
    }

    _g_.m.lockedExt--
    dounlockOSThread()
}

func dounlockOSThread() {
    _g_ := getg()
    if _g_.m.lockedInt != 0 || _g_.m.lockedExt != 0 {
        return
    }
    _g_.m.lockedg = 0
    _g_.lockm = 0
}
```

函数执行的过程与 runtime.LockOSThread 正好相反。在多数的服务中，我们都用不到这一对函数，不过使用 CGO 或者经常与操作系统打交道的读者可能会见到它们的身影。
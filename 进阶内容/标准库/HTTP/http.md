# HTTP
超文本传输协议（Hypertext Transfer Protocol、HTTP 协议）是今天使用最广泛的应用层协议。多数的编程语言都会在标准库中实现 HTTP/1.1 和 HTTP/2.0 已满足工程师的日常开发需求，今天要介绍的 Go 语言的网络库也实现了这两个大版本的 HTTP 协议。

## 设计原理
HTTP协议是应用层协议，在通常情况下我们都会使用 TCP 作为底层的传输层协议传输数据包，但是 HTTP/3 在 UDP 协议上实现了新的传输层协议 QUIC 并使用 QUIC 传输数据，这也意味着 HTTP 既可以跑在 TCP 上，也可以跑在 UDP 上。

[图-HTTP 与传输层协议](https://img.draveness.me/2020-05-18-15897352888395-http-and-transport-layer.png)

Go 语言标准库通过 net/http 包提供 HTTP 的客户端和服务端实现，在分析内部的实现原理之前，我们先来了解一下 HTTP 协议相关的一些设计以及标准库内部的层级结构和模块之间的关系。

### 请求和响应

HTTP 协议中最常见的概念是 HTTP 请求与响应，我们可以将它们理解成客户端和服务端之间传递的消息，客户端向服务端发送 HTTP 请求，服务端收到 HTTP 请求后会做出计算后以 HTTP 响应的形式发送给客户端。

[HTTP 请求与响应](https://img.draveness.me/2020-05-18-15897352888407-http-request-and-response.png)

与其他的二进制协议不同，作为文本传输协议，HTTP 协议的协议头都是文本数据，HTTP 请求头的首行会包含请求的方法、路径和协议版本，接下来是多个 HTTP 协议头以及携带的负载。

```
GET / HTTP/1.1
User-Agent: Mozilla/4.0 (compatible; MSIE5.01; Windows NT)
Host: draveness.me
Accept-Language: en-us
Accept-Encoding: gzip, deflate
Content-Length: <length>
Connection: Keep-Alive

<html>
    ...
</html>

```
HTTP 响应也有着比较类似的结构，其中也包含响应的协议版本、状态码、响应头以及负载，在这里就不展开介绍了。

### 消息边界
HTTP 协议主要还是跑在TCP协议上，TCP协议时面向连接的、可靠的、基于字节流的传输层通信协议，应用层交给TCP协议的数据并不会以消息为单位向目的主机传输，这些数据在某些情况下会被组合成一个数据段发送给目标主机。因为TCP协议时基于字节流的，所以基于 TCP 协议的应用层协议都需要自己划分消息的边界。

[图-实现消息边界的方法](https://img.draveness.me/2020-05-18-15897352888414-message-framing.png)

在应用层协议中，最常见的两种解决方案就是基于长度或者基于终结符（Delimiter）。HTTP 协议其实同时实现了上述两种方案，在多数情况下 HTTP 协议都会在协议头中加入 Content-Length 表示负载的长度，消息的接收者解析到该协议头之后就可以确定当前 HTTP 请求/响应结束的位置，分离不同的 HTTP 消息，下面就是一个使用 Content-Length 划分消息边界的例子：
```
HTTP/1.1 200 OK
Content-Type: text/html; charset=UTF-8
Content-Length: 138
...
Connection: close

<html>
  <head>
    <title>An Example Page</title>
  </head>
  <body>
    <p>Hello World, this is a very simple HTML document.</p>
  </body>
</html>
```
不过 HTTP 协议除了使用基于长度的方式实现边界，也会使用基于终结符的策略，当 HTTP 使用块传输（Chunked Transfer）机制时，HTTP 头中就不再包含 Content-Length 了，它会使用负载大小为 0 的 HTTP 消息作为终结符表示消息的边界。

### 层级结构

Go 语言的 net/http 中同时包好了 HTTP 客户端和服务端的实现，为了支持更好的扩展性，它引入了 net/http.RoundTripper 和 net/http.Handler 两个接口。

net/http.RoundTripper 是用来表示执行HTTP请求的接口，调用方将请求作为参数可以获取请求对应的响应，而 net/http.Handler 主要用于 HTTP 服务器响应客户端的请求：

```
type RoundTripper interface {
    RoundTrip(*Request) (*Response, error)
}
```

HTTP请求的接受方（server端）可以实现 net/http.Handler接口，其中实现了处理HTTP请求的逻辑，处理的过程中会调用 net/http.ResponseWriter接口的方法构造 HTTP响应，它提供的三个接口 Header、Write和 WriteHeader 分别会获取HTTP响应、将数据写入负载（response 的主体部分）以及写入响应头（设置 status code，默认设置为http.StatusOK，就是200的状态码。）：

```
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}

type ResponseWriter interface {
    Header() Header
    Write([]byte) (int, error)
    WriteHeader(statusCode int)
}
```
也就是说，实现了ServeHTTP方法的都是Handler。注意ServeHTTP方法的参数：http.ResponseWriter接口和Request指针。
在Handler的注释中，给出了几点主要的说明：

1. Handler 用于响应一个HTTP request
2. 接口中唯一的方法 ServeHTTP 应该用来将 response header和需要响应的数据写入到 ResponseWriter 中，然后返回。返回意味这个请求已经处理结束，不能再使用这个 ResponseWriter、不能再从 Request.Body 中读取数据，不能并发调用已完成的 ServeHTTP 方法
3. handler应该先读取 Request.Body，然后再写 ResponseWriter。只要开始向 ResponseWriter写数据后，就不能再从 Request.Body中读取数据。
4. handler只能用来读取request的body，不能修改已取得的Request(因为它的参数Request是指针类型的)

很显然，ResponseWriter的作用是构造响应header，并将响应header和响应数据通过网络链接发送给客户端。

客户端和服务端面对的都是双向的HTTP请求与响应，客户端构建请求时并等待着响应，服务器处理请求并返回响应。
HTTP 请求和响应在标准库中不止有一种实现，它们都包含了层级结构，标准库中的 net/http.RoundTripper 包含如下所示的层级结构：

[图-HTTP标准库的层级结构]
每个 net/http.RoundTripper 接口的实现都包含了一种向远程发送请求的过程；标准库中也提供了多种 net/http.Handler 的实现为客户端 HTTP 请求提供不同的服务。

## 客户端
客户端可以直接通过 net/http.Get 函数使用默认的客户端 net/http.DefaultClient 发起 HTTP 请求，也可以自己构建新的 net/http.Client 实现自定义的事务，在多数情况下使用默认的客户端都能满足我们的需求，除了自定义的HTTP事务之外，我们还可以实现自定义的 net/http.CookieJar 接口来管理和使用HTTP的Cookie：

[图-事务和Cookie]

事务和Cookie 是在HTTP客户端包为我们提供的两个重要的模块，我们将从 http get 请求开始，按照构建请求、数据传输、获取连接以及等待响应几个模块分析客户端的实现原理。当我们调用 net/http.Client.Get 方法发出 http请求时，会按照如下的步骤执行：

1. 调用 net/http.NewRequest 根据方法名、URL和请求体构建请求
2. 调用 net/http.Transport.RoundTrip 开启 HTTP 事物、获取连接并发送请求
3. 在 HTTP 持久连接的 net/http.persistConn.writeLoop 方法中等待响应

[图-客户端的几大结构体]

HTTP的客户端包含几个比较重要的结构体，它们分别是 net/http.Client、net/http.Transport 和 net/http.persistConn：

* net/http.Client 是 HTTP 客户端，它的默认值是使用 net/http.DefaultTransport 的 HTTP 客户端
* net/http.Transport 是 net/http.RoundTrip 接口的实现，它主要的作用就是支持HTTP/HTTPS 请求和 HTTP 代理
* net/http.persistConn 封装了一个 TCP 的持久链接，是我们与远程交换消息的句柄（Handle）

客户端 net/http.Client 是级别较高的抽象，它提供了 HTTP 的一些细节，包括 Cookies 和重定向；而 net/http.Transport 结构体会处理 HTTP/HTTPS 协议的底层实现细节，其中包含连接重用、构建请求以及发送请求等功能

### 构建请求

net/http.Request 结构体表示 HTTP 服务接受到的请求或者是 HTTP 客户端发出的请求，其中包含HTTP请求的方法、URL、协议版本、协议头以及请求体等字段，除了这些字段之外，它还会持有一个指向HTTP响应的引用：

```
type Request struct {
  Method  string
  URL *url.URL

  Proto string
  ProtoMajor int
  ProtoMinor int

  Header Header
  Body io.ReadCloser
  
  ...
  Response *Response
}
```

net/http.NewRequest 是标准库提供的用于创建请求的方法，这个方法会对HTTP请求的字段进行校验并根据输入的参数拼装成新的请求结构体。

```
func NewRequestWithContext(ctx context.Context, method, url string, body io.Reader) (*Request, error) {
  if method == "" {
      method = "GET"
  }

  if !validMethod(method) {
      return nil, fmt.Errorf("net/http: invalid method %q", method)
  }

  u, err := urlpkg.Parse(url)
  if err != nil {
      return nil, err
  }

  rc, ok := body.(io.ReaderCloser)
  if !ok && body != nil {
    rc = ioutil.NopCloser(body)
  }

  u.Host = removeEmptyPort(u.Host)
  req := &Request{
    ctx: ctx,
    Method: method,
    URL: u,
    Proto: "HTTP/1.1"
    ProtoMajor: 1,
    ProtoMinor: 1,
    Header: make(Header),
    Body: rc,
    Host: u.Host,
  }

  if body != nil {
    ...
  }

  return req, nil
}
```

请求拼装的过程比较简单，它会检查对输入的方法、URL以及负载，对他们进行校验并初始化了新的 net/http.Request 结构体，处理负载的过程稍微有一些复杂，我们会根据负载的类型不同，使用不同的方法将它们包装成 io.ReadCloser 类型。

### 开启事务

当我们使用标准库构建了 HTTP 请求之后，就会开启HTTP事务发送HTTP请求并等待远程的响应，经过下面一连串的函数调用，我们最终来到了标准库实现底层 HTTP 协议的结构体 — 
net/http.Transport：

1. net/http.Client.Do
2. net/http.Client.do
3. net/http.Client.send
4. net/http.send
5. net/http.Transport.RoundTrip

net/http.Transport 实现了 net/http.Transport.RoundTrip 接口，也是整个请求过程中最重要的并且最复杂的结构体，该结构体会在 net/http.Transport.roundTrip 方法中发送 HTTP 请求并等待响应，那我们可以将该函数执行过程分成两个部分：

* 根据URL的协议查找并执行自定义的 net/http.RoundTripper 实现；
* 从连接池中获取或者初始化新的持久连接并调用连接的 net/http.persistConn.roundTrip 方法发出请求；

我们可以在标准库的 net/http.Transport 中调用 net/http.Transport.RegisterProtocol 方法为不同的协议注册 net/http.RoundTripper 的实现，在下面这段代码中就会根据 URL 中的协议选择对应的实现来替代默认的逻辑：

```
// Transport.roundTrip是主入口，它通过传入一个request参数，由此选择一个合适的长连接来发送该request并返回response。
func (t *Transport) roundTrip(req *Request) (*Response, error) {
  ctx := req.Context()
  scheme := req.URL.Scheme
  
  ...
  if altRT := t.alternateRoundTripper(req); altRT != nil {
    if resp, err := altRT.RoundTrip(req); err != ErrSkipAltProtocol {
        return resp, err
    }
  }
}
```

在默认情况下我们会使用net/http.persistConn 持久连接处理 HTTP 请求，该方法会先获取用于发送请求的连接，随后调用 net/http.persistConn.roundTrip 方法：

```
func (t *Transport) roundTrip(req *Request) (*Response, error) {
    ...
    for {
      select {
        case <-ctx.Done():
            req.closeBody()
            return nil, ctx.Err()
        default:
      }

      treq := &transportRequest{Request: req, trace: trace}

      // connectMethodForRequest函数通过输入一个request返回一个connectMethod(简称cm)，该类型通过
      // {proxyURL,targetScheme,tartgetAddr,onlyH1},即{代理URL，server端的scheme，server的地址，是否HTTP1}
      // 来表示一个请求。一个符合connectMethod描述的request将会在Transport.idleConn中匹配到一类长连接。

      // cm：符合connectMethod描述的request类型
      cm, err := t.connectMethodForRequest(treq)

      if err != nil {
        return nil, err
      }

      pconn, err := t.getConn(treq, cm)
      if err != nil {
        return nil, err
      }

      resp, err := pconn.roundTrip(treq)
      if err != nil {
        return resp, nil
      }
    }
}
```
net/http.Transport.getConn 是获取连接的方法，该方法会通过两种方式获取用于发送请求的连接：

```
func (t *Transport) getConn(treq *transportRequest, cm connectMethod) (pc *persistConn, err error) {
    req := treq.Request
    ctx := req.Context()
    
    w := &wantConn{
      cm: cm,
      key: cm.key(),
      ctx: ctx,
      ready: make(chan struct{}, 1),
    }

    if delivered := t.queueForIdleConn(w); delivered {
      return w.pc, nil
    }

    t.queueForDial(w)

    select {
    case <-w.ready:
        ...
        return w.pc, w.err
    ...
    }
}
```
1. 调用 net/http.Transport.queueForIdleConn 在队列中等待闲置的连接
2. 调用 net/http.Transport.queueForDial 在队列中等待建立新的连接

连接是一种相对比较昂贵的资源，如果在每次发出 HTTP 请求之前都建立新的连接，可能会消耗比较多的时间，带来较大的额外开销，通过连接池对资源进行分配和复用可以有效的提高 HTTP 请求的整体性，多数的网络客户端都会采用类型的策略来复用资源。

当我们调用 net/http.Transport.queueForDial 方法尝试 与远程建立连接，标准库会在内部启动新的 Goroutine 执行 net/http.Transport.dialConnFor 用于建立连接，从最终调用的 net/http.Transport.dialConn 方法中我们能找到TCP连接和 net 库的身影：

```
func (t *Transport) dialConn(ctx context.Context, cm connectMethod) (pconn *persistConn, err error) {
    pconn = &persistConn{
      t: t,
      cacheKey: cm.key()
      reqch: make(chan requestAndChan, 1),
      writech: make(chan writeRequest, 1),
      closech: make(chan struct{}),
      writeErrCh: make(chan error, 1),
      writeLoopDone: make(chan struct{}),
    }

    conn, err := t.dial(ctx, "tcp", cm.addr())
    if err != nil {
      return nil, err
    }

    pconn.conn = conn

    pconn.br = bufio.NewReaderSize(pconn, t.readBufferSize())
    pconn.bw = bufio.NewWriterSize(persistConnWriter{pconn}, t.writeBufferSize())

    go pconn.readLoop()
    go pconn.writeLoop()
    return pconn, nil
}
```

在创建新的TCP连接后，我们还会在后台为当前的连接创建两个 Goroutine,分别从TCP连接中读取数据或者向TCP 写入数据，从建立连接的过程我们就可以发现，如果我们为每一个HTTP 请求都创建新的连接并启动 Goroutine 处理读写数据，会占用很多的资源。

### 等待请求 
持久的TCP连接会实现 net/http.persistConn.roundTrip 方法处理写入HTTP请求并 select 语句中等待响应的返回：

```func (pc *persistConn) roundTrip(req *transportRequest) (resp *Response, err error) {
    writeErrCh := make(chan error, 1)
    pc.writech <- writeRequest{req, writeErrCh, continueCh}

    resc := make(chan responseAndError)
    pc.reqch <- requestAndChan{
        req: req.Request,
        ch: resc,
    }

    for {
        select {
        case re := <-resc:
            if re.err != nil {
                return nil, pc.mapRoundTripError(req, startByteWritten, re.err)
            }
            return re.res, nil
        ...
        }
    }
}
```
每个 HTTP请求都是由另一个Goroutine中的 net/http.persistConn.writeLoop 循环写入的，这两个 Goroutine 独立的执行并通过 Channel 进行通信。net/http.Request.write 方法会根据 net/http.Request 结构体中的字段按照 HTTP 协议组成 TCP数据段：

```
func (pc *persistConn) writeLoop() {
    defer close(pc.writeLoopDone)

    for {
        select {
        case wr := <-pc.writech:
            startByteWritten := pc.nwrite
            wr.req.Request.write(pc.bw, pc.isProxy, wr.req.extra, pc.waitForCountinue(wr.continueCh))
            ...
        case <-pc.closech:
            return
        }
    }
}
```

当我们调用 net/http.Request.write 方法向请求中写入数据时，实际上就直接写入了 net/http.persistConnWriter 中的 TCP 连接中，TCP 协议栈会负责将HTTP请求的内容发送到目标机器上：

```
type persistConnWriter struct {
  pc *persistConn
}

func (w persistConnWriter) Write(p []byte) (n int, err error) {
    n, err := w.pc.conn.Write(p)
    w.pc.nwrite += int64(n)
    return
}
```

持久连接中的另一个读循环 net/http.persistConn.readLoop 会负责从 TCP 连接中读取数据并将数据发送给 HTTP 请求的调用方，真正解析 HTTP 协议的还是 net/http.ReadResponse 方法：

```
func ReadResponse(r *bufio.Reader, req *Request) (*Response, error) {
    tp := textproto.NewReader(r)
    resp := &Response{
        Request: req,
    }

    line, _ := tp.ReadLine()
    if i := strings.IndexByte(line, ' '); i == -1 {
        return nil, badStringError("malformed HTTP response", line)
    } else {
        resp.Proto = line[:1]
        resp.Status = strings.TrimLeft(line[i+1:], " ")
    }

    statusCode := resp.Status
    if i := strings.IndexByte(resp.Status, ` `); i != -1 {
        statusCode = resq.Status[:i]
    }
    resq.StatusCode, err := strconv.Atoi(statusCode)

    resq.ProtoMajor, resq.ProtoMinor, _ = ParseHTTPVersion(resp.Proto)

    mimeHeader, _ := tp.ReadMIMEHeader()
    resq.Header = Header(mimeHeader)

    readTransfer(resp, r)
    return resp, nil
}
```

我们在上述方法中可以看到HTTP响应结构的大致框架，其中包状态码、协议版本、请求头、等内容，响应体还是在读取循环 net/http.persistConn.readLoop 中根据HTTP 协议头进行解析的。

## 服务器

Go语言标准库 net/http 包提供了非常易用的接口，如下图所示，我们可以利用标准库提供的功能快速搭建新的HTTP 服务：

```
func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func main() {
    http.HandleFunc("/", handler)
    log.Fatal(http.ListenAndServer(":8080", nil))
}
```

上述的 main 函数只调用了两个标准库提供的函数，他们分别是用于注册处理器的 net/http.HandleFunc 函数和用于监听和处理请求的 net/http.ListenAndServer，多数的服务器框架都会包含这两类接口，分别负责注册处理器和处理外部请求。

### 注册处理器

HTTP服务是一组实现了 net/http.Handler 接口的处理器组成的，处理HTTP请求时会根据请求的路由选择合适的处理器：

[图-HTTP服务与处理器]

当我们直接调用 net/http.HandleFunc 注册处理器时，标准库时会使用默认的 HTTP 服务器 net/http.DefaultServeMux 处理请求，该方法会直接调用 net/http.ServeMux.HandleFunc：

```
func (mux *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
    mux.Handle(pattern, HandlerFunc(handler))
}
```

上述方法会将处理器转换成 net/http.Handler 接口类型调用 net/http.ServeMux.Handle 注册处理器：

```
func (mux *ServeMux) Handle(pattern string, handler Handler) {

    // 检查该路由是否已经注册过，
    if _, exist := mux.m[pattern]; exist {
        panic("http: multiple registrations for " + pattern)
    }

    // 实例化一个 muxEntry
    e := muxEntry{h: handler, pattern: pattern}
    // 将该路由与该 MuxEntry 的实例保存 mux.m 中
    mux.m[pattern] = e

    // 如果该路由以 "/" 结尾
    // 就把该路由按照大到小的路径长度插入到 mux.e 中
    if pattern[len(pattern)-1] == '/' {
        mux.es = appendSorted(mux.es, e)
    }

    // 如果该路由不以 "/"，标记该 mux 中有路由的路径带有主机名
    if pattern[0] != '/' {
        mux.hosts = true
    }
} 
```

路由和对应的处理器会被组成 net/http.DefaultServeMux 会持有一个 net/http.muxEntry 哈希，其中存储了从 URL到处理器的映射关系，HTTP 服务器在处理请求时使用该哈希查找处理器。

### 处理请求

标准库提供的 net/http.ListenAndServe 函数可以用来监听TCP连接并处理请求，该函数会使用传入的监听地址和处理器初始化一个 HTTP 服务器 net/http.Server，调用该服务器的 net/http.Server.ListenAndServe 方法：

```
func ListenAndServe (addr string, handler Handler) error {
    server := &Server{Addr: addr, Handler: handler}
    return server.ListenAndServe()
}
```

net/http.Server.ListAndServe 方法会使用网络库 net.Listen 函数监听对应地址的上的TCP连接并通过net/http.Server.Serve 处理客户端的请求：

```
func (srv *Server) ListenAndServe() error {
    if addr == "" {
        addr = ":http"
    }

    ln, err := net.Listen("tcp", addr)
    if err != nil {
        return err
    }

    return srv.Serve(ln)
}
```

net/http.Server.serve 会在循环中监听外部的TCP连接并为每个连接调用 net/http.Server.newConn 创建新的结构体 net/http.conn，它是HTTP 连接的服务端表示：

```
func (srv *Server) Serve(l net.Listener) error {
    l = &onceCloseListener{Listen: l}
    defer l.Close()

    // Background 方法返回一个非nil，空的上下文片段
    // 这个空的Context一般用于整个Context树的根节点。然后我们使用context
    // WithCancel(parent)函数，创建一个可取消的子Context，然后当作参数传给
    // goroutine使用，这样就可以使用这个子Context跟踪这个goroutine。
    baseCtx := context.Backgroud()

    // 通过Context我们也可以传递一些必须的元数据，这些数据会附加在Context上以供使用。
    ctx := context.WithValue(baseCtx, ServerContextKey, srv)
    for {
        rw, err := l.Accept()
        if err != nil {
            select {
            case <-srv.getDoneChan():
                return ErrServerClosed
            default:
            }

            ...
            return err
        }

        connCtx := ctx
        c := srv.newConn(rw)
        c.setState(c.rwc, StateNew)     // before Serve can return
        go c.serve(connCtx)
    }

}
```
创建了服务端的连接之后，标准库的实现会为每个 HTTP请求创建单独的 Goroutine 并在其中调用 net/http.Conn.serve 方法，如果当前的 HTTP 服务接受到了海量的请求，会在内部创建大量的 Goroutine，这可能会使整个服务质量明显降低无法处理请求。

```
func (c *conn) serve(ctx context.Context) {
    c.remoteAddr = c.rwc.RemoteAddr().String()

    ctx = context.WithValue(ctx, LocalAddrContextKey, c.rwc.LocalAddr())
    ctx, cancelCtx := context.WithCancel(ctx)
    c.cancelCtx = cancelCtx
    defer cancelCtx()

    c.r = &connReader{conn: c}
    c.bufr = newBufioReader(c, r)
    c.bufw = newBufioWriterSize(checkConnErrorWriter{c}, 4 << 10)

    for {
      w, _ := c.readRequest(ctx)
      serverHandler{c.server}.ServeHTTP(w, w.req)
      w.finishRequest()
    }
}
```
上述代码片段是我们简化后的连接处理过程，其中包含读取HTTP请求、调用 Handler 处理HTTP 请求以及调用完成该请求。读取HTTP请求会调用 net/http.Conn.readRequest方法，该方法会从连接中获取一个HTTP请求并构建一个实现了 net/http.ResponseWriter 接口的变量 net/http.response， 向该结构体写入的数据都会被转发到它持有的缓冲区中：

```
func (w *response) write(lenData int, dataB []byte, dataS string) (n int, err error) {
    ...
    w.written += int64(lenData)
    if w.contentLength != -1 && w.written > w.contentLength {
        return 0, ErrContentLength
    }
    if DataB != nil {
        return w.w.Write(dataB)
    } else {
        return w.w.WriteString(dataS)
    }
}
```

解析了 HTTP 请求并初始化 net/http.ResponseWriter 之后，我们就可以调用 net/http.serverHandler.ServeHTTP 方法查找处理器来处理HTTP请求了：
```
type serverHandler struct {
    srv *Server
}

func (sh serverHandler) ServeHTTP(rw ResponseWriter, req *Request) {
    handler := sh.srv.Handler
    if handler == nil {
        handler = DefaultServeMux
    }

    if req.RequestURI == "*" && req.Method == "OPTIONS" {
        handler = globalOptionsHandler{}
    }

    handler.ServeHTTP(rw, req)
}
``` 
如果当前的HTTP服务器中不包含任何处理器，我们会使用默认的 net/http.DefaultServeMux 处理外部的HTTP请求。

net/http.ServeMux 是一个 HTTP 请求的多路复用器，它可以接受外部的HTTP请求、根据请求的URL匹配并调用最合适的处理器：

```
func (mux *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
    h, _ := mux.Handler(r)
    h.ServeHTTP(w, r)
}
```

经过一系列的函数调用，上述过程最终会调用 HTTP 服务器的 net/ServeMux.math 方法，该方法会遍历前面注册过的路由表并根据特定规则进行匹配：
```
type ServeMux struct {
   mu sync.RWMutex

   // m是用来存储路由与处理函数映射关系map，但 ServeMux 为了方便
   // map存放的值其实是放有处理函数和路由路径的 muxEntry 结构体
   m map[string]muxEntry
   // es 按照路由长度从长到小短的存放路由的切片
   es []muxEntry
   // 标记路由中是否带有主机名
    hosts bool
}

type muxEntry struct {
    h Handler   // 处理函数
    pattern string  // 路由路径
}

func (mux *ServeMux) match(path string) (h Handler, pattern string) {
   v, ok := mux.m[path]
   if ok {
      return v.h v.pattern
   }

   for _, e := range mux.es {
      if strings.HasPrefix(path, e.pattern) {
          return e.h， e.pattern
      }
   }

   return nil, ""
}
```

ServeMux 暴露的方法主要是：
```
func (mux *ServeMux) Handle(pattern string, handler Handler)
func (mux *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request))
func (mux *ServeMux) Handler(r *Request) (h Handler, pattern string)
func (mux *ServeMux) ServeHTTP
```

```
package main

import (
	"fmt"
	"net/http"
)

func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello\n")
}

func world(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "World\n")
}

func main() {
	server := http.Server{
		Addr: "127.0.0.1:8080",
	}
	http.HandleFunc("/hello", hello)
	http.HandleFunc("/world", world)

	server.ListenAndServe()
}
```


HandleFunc 方法接受一个具体的处理函数将其包装成 Handler（它使用第二个参数的函数作为 handler，处理匹配到的url路径请求），不难看出，HandleFunc() 使得我们可以直接使用一个函数作为 handler，而不需要自定义一个实现 Handler 接口的类型。正如上面实例，并没有定义 Handler 类型，也没有去实现 ServeHTTP() 方法，而是直接定义函数， 并将其作为 handler。

换句话说，HandleFunc() 使得我们可以更简便的为某些 url 路径注册 handler。但是，使用 Handlefunc() 毕竟时图简便，有时候不得不使用 Handle()，比如我们确定要定义一个type。


* Handle() 和 HandleFunc() 是函数，用来给 url 绑定 handler.
* Handler 和 HandleFunc类型，用来处理请求。

接下来看 Handle()、HandleFunc() 以及 Handler、HandlerFunc的定义就很清晰了：

```
func Handle(pattern string, handler Handler) {}
func HandleFunc(pattern handler func(RequestWriter, *Request)) {}

type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}
type HandlerFunc func(ResponseWriter, *Request)
func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request)
```
Handle()和 handleFunc() 都是为某个 url路径模式绑定一个对应的 handler，只不过 HandleFunc() 是直接使用函数作为 handler，而 handle() 是使用Handler类型的示例作为handler。

Handler接口类型的实例都实现了 ServeHTTP() 方法，都用来请求并响应给客户端。

HandlerFunc类型不是接口，但它有一个方法 ServeHTTP()，也就是说 HandlerFunc其实也是一种Handler。

因为 HandlerFunc 是类型，只要某个函数的签名是 func(ResponseWriter, *Request)，它就是HandlerFunc类型的一个实例。另一方面，这个类型的实例（可能时参数、可能时返回值）可以和某个签名为 func(ResponseWriter, *Request) 的函数进行相互赋值。这个过程可能很隐式，但经常出现相关的用法。
例如：
```
// 一个函数类型的 handler
func myhf(ResponseWriter, *Request) {
}

// 以 HandlerFunc 类型作为参数类型
func a(hf HandlerFunc) {}

// 所以可以将 myhf作为a() 的参数
a(myhf)
```

如果请求的路径和路由中的表项匹配成功，我们会调用表项中对应的处理器，处理器中包含的业务逻辑会通过 net/http.ResponseWriter 构建HTTP 请求并通过 TCP 连接发送回客户端。
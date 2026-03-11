# 标准 TCP 编程详解

本文档详细介绍 `cmd/tcp/` 目录下的标准 TCP Socket 编程示例。

## 概述

本模块使用 Go 标准库 `net` 包实现了一个简单的 TCP 回显（Echo）服务器和客户端。这是网络编程的基础入门示例，展示了 TCP 连接的建立、数据传输和并发处理。

## 文件说明

| 文件 | 说明 |
|------|------|
| `cmd/tcp/service.go` | TCP 回显服务端 |
| `cmd/tcp/client.go` | TCP 回显客户端 |

---

## 服务端详解 (`service.go`)

### 工作流程

```
启动监听 (net.Listen)
    │
    ├──→ 等待客户端连接 (listen.Accept)
    │        │
    │        └──→ 启动 goroutine 处理连接 (go process)
    │                 │
    │                 ├──→ 读取客户端数据 (reader.Read)
    │                 ├──→ 打印接收到的数据
    │                 └──→ 回显数据给客户端 (conn.Write)
    │
    └──→ 继续等待下一个连接（循环）
```

### 核心代码解析

#### 1. 启动 TCP 监听

```go
listen, err := net.Listen("tcp", "127.0.0.1:888")
```

- `net.Listen("tcp", addr)` 在指定地址上创建一个 TCP 监听器
- `"127.0.0.1:888"` 表示只监听本地回环地址的 888 端口
- 返回一个 `net.Listener` 接口

#### 2. 接受客户端连接

```go
for {
    conn, err := listen.Accept()  // 阻塞等待连接
    go process(conn)              // 启动 goroutine 并发处理
}
```

- `Accept()` 是一个阻塞调用，直到有新的客户端连接
- 每个连接都在独立的 goroutine 中处理，实现并发

#### 3. 处理客户端数据

```go
func process(conn net.Conn) {
    defer conn.Close()
    for {
        reader := bufio.NewReader(conn)
        var buf [128]byte
        n, err := reader.Read(buf[:])
        recvStr := string(buf[:n])
        fmt.Println("收到client端发来的数据：", recvStr)
        conn.Write([]byte(recvStr))  // 回显
    }
}
```

- 使用 `bufio.NewReader` 包装连接以提供缓冲读取
- 读取最多 128 字节的数据
- 将接收到的数据原样返回（回显）
- 使用 `defer conn.Close()` 确保连接在函数结束时关闭

---

## 客户端详解 (`client.go`)

### 工作流程

```
建立连接 (net.Dial)
    │
    ├──→ 发送数据 (con.Write)
    ├──→ 接收回显 (con.Read)
    ├──→ 打印响应
    ├──→ 等待 5 秒 (time.Sleep)
    └──→ 循环发送
```

### 核心代码解析

#### 1. 建立 TCP 连接

```go
con, err := net.Dial("tcp", "127.0.0.1:888")
defer con.Close()
```

- `net.Dial("tcp", addr)` 主动发起 TCP 连接
- 返回一个 `net.Conn` 接口，用于后续的读写操作

#### 2. 数据收发循环

```go
for {
    con.Write([]byte("hello, zhangsan"))  // 发送数据
    buf := [512]byte{}
    n, err := con.Read(buf[:])            // 接收回显
    fmt.Println(string(buf[:n]))          // 打印响应
    time.Sleep(5 * time.Second)           // 间隔 5 秒
}
```

- 每次发送固定字符串 `"hello, zhangsan"`
- 使用 512 字节缓冲区接收服务端响应
- 每次发送间隔 5 秒

---

## 运行示例

### 启动服务端

```bash
go run cmd/tcp/service.go
```

### 启动客户端（另一个终端）

```bash
go run cmd/tcp/client.go
```

### 预期输出

**服务端输出：**
```
收到client端发来的数据： hello, zhangsan
收到client端发来的数据： hello, zhangsan
...
```

**客户端输出：**
```
hello, zhangsan
hello, zhangsan
...
```

---

## 涉及的 Go 标准库

| 包 | 函数/类型 | 说明 |
|----|-----------|------|
| `net` | `Listen(network, address)` | 创建 TCP/UDP 监听器 |
| `net` | `Dial(network, address)` | 发起 TCP/UDP 连接 |
| `net` | `Conn` 接口 | 表示一个网络连接，提供 Read/Write 方法 |
| `net` | `Listener` 接口 | 表示一个网络监听器，提供 Accept 方法 |
| `bufio` | `NewReader(rd)` | 创建带缓冲的读取器 |

---

## 关键概念

### TCP 三次握手

客户端调用 `net.Dial` 时，Go 标准库自动完成 TCP 三次握手：

```
客户端                      服务端
  │                          │
  │──── SYN ────────────────→│  第1步：客户端发送 SYN
  │                          │
  │←─── SYN+ACK ────────────│  第2步：服务端回复 SYN+ACK
  │                          │
  │──── ACK ────────────────→│  第3步：客户端发送 ACK
  │                          │
  │    连接建立，开始传输数据   │
```

### goroutine 并发模型

```
主 goroutine (监听循环)
  │
  ├──→ goroutine 1: 处理客户端 A
  ├──→ goroutine 2: 处理客户端 B
  ├──→ goroutine 3: 处理客户端 C
  └──→ ...
```

每个客户端连接由独立的 goroutine 处理，Go 的调度器自动管理这些轻量级线程的调度，无需手动管理线程池。

---

## 扩展思考

1. **错误处理**：当前示例的错误处理比较简单，生产代码应增加重试机制和日志记录
2. **优雅关闭**：可以通过 `context` 包实现服务端的优雅关闭
3. **超时控制**：使用 `conn.SetDeadline()` 设置读写超时
4. **连接池**：对于高并发场景，可以使用连接池复用连接

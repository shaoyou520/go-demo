# 原始套接字编程详解

本文档详细介绍 `cmd/sock_raw/` 和 `pkg/tcp/` 目录下的原始套接字（Raw Socket）编程示例。

## 概述

原始套接字允许程序直接在 IP 层收发数据包，绕过操作系统内核的 TCP/UDP 协议栈。这使得程序可以：

- 捕获和分析经过本机的所有网络数据包
- 构造自定义的 IP/TCP 头部
- 实现自定义协议
- 进行网络诊断和监控

> **注意**：原始套接字操作需要 root 权限。

## 文件说明

| 文件 | 说明 |
|------|------|
| `cmd/sock_raw/service.go` | 原始 IPv4 数据包监听器 |
| `cmd/sock_raw/client.go` | 原始 IPv4 数据包发送器 |
| `pkg/tcp/my_socket.go` | 原始套接字监听封装库 |

---

## 原始套接字基础知识

### 套接字类型对比

```
应用层 Socket（常规编程使用）
┌─────────────────────────────┐
│  SOCK_STREAM (TCP)          │  ← net.Dial("tcp", addr)
│  SOCK_DGRAM  (UDP)          │  ← net.Dial("udp", addr)
└─────────────┬───────────────┘
              │ 操作系统自动处理 TCP/UDP/IP 头部
              ▼
传输层 Socket（本项目涉及）
┌─────────────────────────────┐
│  SOCK_RAW + IPPROTO_TCP     │  ← 可以看到 TCP 头部
│  SOCK_RAW + IPPROTO_ICMP    │  ← 可以看到 ICMP 头部
│  SOCK_RAW + IPPROTO_UDP     │  ← 可以看到 UDP 头部
└─────────────┬───────────────┘
              │ 操作系统自动处理 IP 头部
              ▼
网络层 Socket（最底层）
┌─────────────────────────────┐
│  SOCK_RAW + IPPROTO_RAW     │  ← 需要自己构造 IP 头部
└─────────────────────────────┘
```

### syscall.Socket 参数说明

```go
fd, _ := syscall.Socket(domain, sockType, protocol)
```

**第一个参数 `domain`（地址族）：**

| 常量 | 说明 |
|------|------|
| `syscall.AF_INET` | IPv4 网络通信 |
| `syscall.AF_INET6` | IPv6 网络通信 |
| `syscall.AF_UNIX` | 本机进程间通信 |

**第二个参数 `sockType`（套接字类型）：**

| 常量 | 说明 |
|------|------|
| `syscall.SOCK_STREAM` | 基于 TCP 的流式 Socket（应用层） |
| `syscall.SOCK_DGRAM` | 基于 UDP 的数据报 Socket（应用层） |
| `syscall.SOCK_RAW` | 原始套接字，可构造传输层协议头部 |

**第三个参数 `protocol`（协议号）：**

| 常量 | 协议号 | 说明 |
|------|--------|------|
| `syscall.IPPROTO_TCP` | 6 | 接收 TCP 协议数据 |
| `syscall.IPPROTO_UDP` | 17 | 接收 UDP 协议数据 |
| `syscall.IPPROTO_ICMP` | 1 | 接收 ICMP 协议数据 |
| `syscall.IPPROTO_IP` | 0 | 接收所有 IP 数据包 |
| `syscall.IPPROTO_RAW` | 255 | 只能发送，不能接收 |

---

## 数据包监听器详解 (`cmd/sock_raw/service.go`)

### 工作流程

```
解析 IP 地址 (net.ResolveIPAddr)
    │
    ▼
创建原始 IP 连接 (net.ListenIP)
    │
    ▼
包装为 IPv4 原始连接 (ipv4.NewRawConn)
    │
    ▼
循环读取数据包 (ipconn.ReadFrom)
    │
    ├──→ IP 头部 (hdr)
    ├──→ 负载数据 (payload)
    └──→ 控制信息 (controlMessage)
```

### 核心代码解析

```go
// 1. 解析 IP 地址
netaddr, _ := net.ResolveIPAddr("ip4", "127.0.0.1")

// 2. 创建原始 IP 连接，监听 TCP 协议的数据包
conn, _ := net.ListenIP("ip4:tcp", netaddr)

// 3. 包装为 IPv4 原始连接，可以分别读取 IP 头部和负载
ipconn, _ := ipv4.NewRawConn(conn)

// 4. 循环读取数据包
for {
    buf := make([]byte, 1500)
    hdr, payload, controlMessage, _ := ipconn.ReadFrom(buf)
    fmt.Println("ipheader:", hdr, payload, controlMessage)
}
```

**`ipv4.NewRawConn`** 的优势在于它能将读取到的数据包自动分离为：
- `hdr`：解析好的 IPv4 头部结构（`*ipv4.Header`）
- `payload`：IP 层负载（即 TCP/UDP 数据段）
- `controlMessage`：控制信息（如接收时间戳等）

---

## 数据包发送器详解 (`cmd/sock_raw/client.go`)

### 工作流程

```
创建原始套接字 (syscall.Socket)
    │
    ▼
构造目标地址 (syscall.SockaddrInet4)
    │
    ▼
构造 IPv4 头部 (ipv4.Header)
    │
    ▼
序列化并发送 (syscall.Sendto)
```

### 核心代码解析

```go
// 1. 创建原始套接字
fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP)

// 2. 构造目标地址
addr := syscall.SockaddrInet4{
    Port: 0,
    Addr: [4]byte{127, 0, 0, 1},
}

// 3. 构造 IPv4 头部
head := ipv4.Header{
    Version:  4,        // IPv4
    Len:      20,       // IP 头部长度（字节）
    TotalLen: 20,       // 总长度
    TTL:      64,       // 生存时间
    Protocol: 6,        // TCP 协议号
    Dst:      net.IPv4(172, 17, 0, 3),   // 目标 IP
    Src:      net.IPv4(172, 17, 0, 99),  // 源 IP
}

// 4. 序列化并发送
payload, _ := head.Marshal()
syscall.Sendto(fd, payload, 0, &addr)
```

**关键点**：
- 使用 `SOCK_RAW` 创建的套接字可以自行构造传输层头部
- `ipv4.Header` 提供了 IP 头部的结构化表示
- `syscall.Sendto` 直接将数据包发送到网络层

---

## 封装库详解 (`pkg/tcp/my_socket.go`)

### NetSocketICMP - ICMP 数据包监听

```go
func NetSocketICMP(netaddr *net.IPAddr) {
    conn, _ := net.ListenIP("ip4:icmp", netaddr)
    for {
        buf := make([]byte, 1024)
        n, addr, _ := conn.ReadFrom(buf)
        msg, _ := icmp.ParseMessage(1, buf[0:n])
        fmt.Println(n, addr, msg.Type, msg.Code, msg.Checksum)
    }
}
```

**功能**：监听所有 ICMP 数据包（如 ping 请求和响应）

**输出信息**：
- `n`：数据包大小
- `addr`：数据包来源地址
- `msg.Type`：ICMP 类型（8=Echo Request, 0=Echo Reply）
- `msg.Code`：ICMP 代码
- `msg.Checksum`：ICMP 校验和

### NetSocketTCP - TCP 数据包监听

```go
func NetSocketTCP(netaddr *net.IPAddr) {
    conn, _ := net.ListenIP("ip4:tcp", netaddr)
    for {
        buf := make([]byte, 1480)
        n, addr, _ := conn.ReadFrom(buf)
        tcpheader := util.NewTCPHeader(buf[0:n])
        if tcpheader == nil {
            continue
        }
        fmt.Println(n, addr, tcpheader, string(buf[tcpheader.DataOffset*4:n]))
    }
}
```

**功能**：监听所有 TCP 数据包，过滤目标端口为 888 的数据包

**输出信息**：
- `n`：数据包大小
- `addr`：数据包来源地址
- `tcpheader`：解析后的 TCP 头部结构
- 数据负载（从 `DataOffset*4` 字节开始的内容）

**过滤机制**：`util.NewTCPHeader` 仅解析目标端口为 888 的数据包，其他数据包返回 `nil`。

---

## net.ListenIP 协议字符串格式

`net.ListenIP` 的第一个参数格式为 `"ip4:protocol"`，其中 `protocol` 指定要监听的协议：

| 协议字符串 | 说明 |
|-----------|------|
| `"ip4:tcp"` | 监听 TCP 数据包（协议号 6） |
| `"ip4:icmp"` | 监听 ICMP 数据包（协议号 1） |
| `"ip4:udp"` | 监听 UDP 数据包（协议号 17） |

---

## 运行示例

### 运行 TCP 数据包监听器

```bash
# 需要 root 权限
sudo go run cmd/sock_raw/service.go
```

### 运行数据包发送器

```bash
# 需要 root 权限
sudo go run cmd/sock_raw/client.go
```

---

## 数据包处理流程图

```
                    ┌──────────────┐
                    │   物理网卡    │
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │   IP 协议栈   │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
       ┌──────▼──────┐  ┌─▼──┐  ┌─────▼─────┐
       │ IPPROTO_TCP │  │ .. │  │IPPROTO_ICMP│
       └──────┬──────┘  └────┘  └─────┬──────┘
              │                        │
       ┌──────▼──────┐          ┌─────▼──────┐
       │ SOCK_RAW    │          │ SOCK_RAW   │
       │ (本项目)     │          │ (本项目)    │
       └──────┬──────┘          └─────┬──────┘
              │                        │
       ┌──────▼──────┐          ┌─────▼──────┐
       │  TCP 协议栈  │          │ ICMP 处理  │
       │  (内核)      │          │ (内核)      │
       └──────┬──────┘          └────────────┘
              │
       ┌──────▼──────┐
       │ SOCK_STREAM │
       │ (常规应用)   │
       └─────────────┘
```

**原始套接字**（`SOCK_RAW`）在 IP 协议栈之后、传输层协议栈之前拦截数据包，因此可以看到完整的 TCP/ICMP 头部信息。

---

## 常见问题

### 1. 为什么需要 root 权限？

原始套接字可以捕获和伪造任意网络数据包，出于安全考虑，Linux 内核要求创建原始套接字的进程必须具有 `CAP_NET_RAW` 能力，通常需要 root 权限。

### 2. 原始套接字和 tcpdump 的关系？

`tcpdump` 等抓包工具底层也是使用原始套接字（或 BPF/libpcap）来捕获网络数据包的。本项目的示例本质上实现了一个简化版的抓包工具。

### 3. 为什么监听缓冲区大小是 1480/1500？

- 以太网 MTU（最大传输单元）通常为 1500 字节
- IP 头部通常为 20 字节
- 因此 IP 负载最大为 1480 字节

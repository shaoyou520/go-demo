# Go 网络编程实战教程

本项目是一个 Go 语言网络编程学习仓库，涵盖从标准 TCP Socket 编程到底层原始数据包操作，再到 Linux 容器网络隔离的完整知识体系。通过循序渐进的示例代码，帮助开发者深入理解 TCP/IP 协议栈的工作原理。

## 目录

- [项目概览](#项目概览)
- [目录结构](#目录结构)
- [环境要求](#环境要求)
- [快速开始](#快速开始)
- [模块详解](#模块详解)
  - [标准 TCP 编程](#1-标准-tcp-编程入门级)
  - [自定义 TCP 头部](#2-自定义-tcp-头部构造中级)
  - [原始套接字编程](#3-原始套接字编程高级)
  - [网络命名空间与桥接](#4-网络命名空间与桥接高级)
  - [工具库](#5-工具库)
- [学习路线](#学习路线)
- [依赖说明](#依赖说明)

---

## 项目概览

本项目适用于以下场景：

- **学习网络编程**：理解 TCP/IP 协议在标准库抽象之下的实际工作方式
- **数据包分析**：学习如何直接检查和构造网络数据包
- **容器网络原理**：了解 Docker 等容器运行时的网络实现机制（网络命名空间、Linux 桥接、虚拟以太网对）
- **自定义协议开发**：掌握自定义协议头部的构造与解析方法

**适用人群**：学习网络编程的软件工程师、理解容器网络的 DevOps 工程师，以及需要实现自定义网络协议或监控工具的开发者。

---

## 目录结构

```
go-demo/
├── cmd/                        # 可执行示例程序（入口点）
│   ├── tcp/                    # 标准 TCP Socket 示例
│   │   ├── service.go          # TCP 回显服务端
│   │   └── client.go           # TCP 回显客户端
│   ├── sock_raw/               # 原始 IP 套接字示例
│   │   ├── service.go          # 原始 IPv4 数据包监听器
│   │   └── client.go           # 原始 IPv4 数据包发送器
│   └── socket/                 # 自定义 TCP 头部示例
│       ├── my_service.go       # 自定义 TCP 头部解析服务端
│       └── my_client.go        # 自定义 TCP 头部构造客户端
├── pkg/                        # 可复用的库包
│   ├── tcp/                    # 原始套接字监听库
│   │   └── my_socket.go        # NetSocketICMP、NetSocketTCP 函数
│   └── my_bridge_net/          # 网络命名空间与桥接工具
│       ├── main.go             # 完整的网络拓扑构建器
│       └── go.mod              # 独立模块定义
├── util/                       # 共享工具包
│   ├── tcp_util.go             # TCPHeader 结构体、校验和计算
│   └── tcp_util_test.go        # TCP 工具单元测试
├── docs/                       # 文档
│   ├── tcp head.md             # TCP 头部格式参考
│   ├── README_标准TCP编程.md    # 标准 TCP 编程详细文档
│   ├── README_原始套接字.md     # 原始套接字编程详细文档
│   ├── README_自定义TCP头部.md  # 自定义 TCP 头部详细文档
│   ├── README_网络命名空间.md   # 网络命名空间详细文档
│   └── README_工具库.md         # 工具库 API 文档
├── go.mod                      # 主模块定义（Go 1.22）
└── go.sum                      # 依赖校验和
```

---

## 环境要求

- **Go**: 1.22 或更高版本
- **操作系统**: Linux（原始套接字和网络命名空间功能需要 Linux 环境）
- **权限**: 原始套接字相关程序需要 root 权限（`sudo`）
- **系统工具**: `ip`、`iptables`（网络命名空间模块需要）

---

## 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/shaoyou520/go-demo.git
cd go-demo
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 运行测试

```bash
go test ./util/...
```

### 4. 运行标准 TCP 示例

```bash
# 终端1：启动服务端
go run cmd/tcp/service.go

# 终端2：启动客户端
go run cmd/tcp/client.go
```

---

## 模块详解

### 1. 标准 TCP 编程（入门级）

**位置**: `cmd/tcp/`

本模块演示了使用 Go 标准库进行 TCP Socket 编程的基本方法。

#### 服务端 (`service.go`)

- 在 `127.0.0.1:888` 上监听 TCP 连接
- 为每个客户端连接启动独立的 goroutine 处理
- 实现了回显（Echo）功能：接收客户端数据并原样返回

```go
// 核心流程
listen, err := net.Listen("tcp", "127.0.0.1:888")  // 监听端口
conn, err := listen.Accept()                         // 接受连接
go process(conn)                                     // 并发处理
```

#### 客户端 (`client.go`)

- 连接到 `127.0.0.1:888`
- 每隔 5 秒发送一次消息 "hello, zhangsan"
- 接收并打印服务端的回显响应

#### 运行方法

```bash
# 终端1
go run cmd/tcp/service.go

# 终端2
go run cmd/tcp/client.go
```

**核心知识点**: `net.Listen`、`net.Dial`、`net.Conn`、goroutine 并发处理

> 详细文档请参阅 [docs/README_标准TCP编程.md](docs/README_标准TCP编程.md)

---

### 2. 自定义 TCP 头部构造（中级）

**位置**: `cmd/socket/`

本模块演示了如何绕过操作系统的 TCP 协议栈，手动构造 TCP 头部并发送数据包。

#### 客户端 (`my_client.go`)

- 使用 `ip4:tcp` 协议建立原始连接
- 手动构造 TCP 头部（源端口 17663，目标端口 888）
- 计算 TCP 校验和
- 在数据负载中附加中文字符串

```go
// 手动构造 TCP 头部
head := util.TCPHeader{
    Source:      17663,
    Destination: 888,
    SeqNum:      2,
    DataOffset:  5,
    Ctrl:        2,  // SYN 标志
    Window:      0xaaaa,
}
data := head.Marshal()
head.Checksum = util.Csum(data, util.To4byte(local), util.To4byte(remote))
```

#### 服务端 (`my_service.go`)

- 使用原始 IP 套接字监听 TCP 数据包
- 使用 `util.NewTCPHeader` 解析 TCP 头部
- 过滤目标端口为 888 的数据包
- 显示解析后的头部信息和负载内容

#### 运行方法（需要 root 权限）

```bash
# 终端1：启动服务端
sudo go run cmd/socket/my_service.go

# 终端2：启动客户端
sudo go run cmd/socket/my_client.go
```

**核心知识点**: TCP 头部格式、校验和计算、原始 IP 连接、字节序处理

> 详细文档请参阅 [docs/README_自定义TCP头部.md](docs/README_自定义TCP头部.md)

---

### 3. 原始套接字编程（高级）

**位置**: `cmd/sock_raw/` 和 `pkg/tcp/`

本模块演示了在 IP 层直接操作数据包的方法，绕过内核的 TCP 协议栈。

#### 原始 IP 数据包监听 (`cmd/sock_raw/service.go`)

- 使用 `ipv4.NewRawConn` 创建原始连接
- 直接读取 IP 头部、负载和控制信息
- 能看到完整的 IP 层数据

#### 原始 IP 数据包发送 (`cmd/sock_raw/client.go`)

- 使用 `syscall.Socket` 创建原始套接字
- 手动构造 IPv4 头部
- 通过 `syscall.Sendto` 直接发送数据包

```go
// syscall 参数说明
// 第一个参数: AF_INET（IPv4网络通信）、AF_UNIX（本机进程通信）、AF_INET6（IPv6网络通信）
// 第二个参数: SOCK_RAW（原始套接字）、SOCK_STREAM（TCP）、SOCK_DGRAM（UDP）
// 第三个参数: IPPROTO_TCP（TCP协议）、IPPROTO_ICMP（ICMP协议）等
fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP)
```

#### 封装库 (`pkg/tcp/my_socket.go`)

提供了两个封装好的原始套接字监听函数：

| 函数 | 说明 |
|------|------|
| `NetSocketICMP(addr)` | 监听并解析 ICMP 数据包（如 ping 请求/响应） |
| `NetSocketTCP(addr)` | 监听并解析 TCP 数据包，包含完整头部信息 |

#### 运行方法（需要 root 权限）

```bash
# 监听原始 TCP 数据包
sudo go run cmd/sock_raw/service.go
```

**核心知识点**: `syscall.Socket`、`SOCK_RAW`、IPv4 头部构造、`net.ListenIP`

> 详细文档请参阅 [docs/README_原始套接字.md](docs/README_原始套接字.md)

---

### 4. 网络命名空间与桥接（高级）

**位置**: `pkg/my_bridge_net/`

本模块实现了类似 Docker 容器网络的完整网络隔离环境，帮助理解容器网络的底层原理。

#### 网络拓扑

```
┌──────────────────────────────────────────────────────────────────────┐
│  主机 A (192.168.0.3)                主机 B (192.168.0.2)            │
│  ┌─────────────────────┐             ┌─────────────────────┐        │
│  │  ns0 (命名空间)      │             │  ns0 (命名空间)      │        │
│  │  nseth0: 10.0.1.1   │             │  nseth0: 10.0.2.1   │        │
│  └────────┬────────────┘             └────────┬────────────┘        │
│           │ veth pair                         │ veth pair            │
│  ┌────────┴────────────┐             ┌────────┴────────────┐        │
│  │  bridge0            │             │  bridge0            │        │
│  │  10.0.1.254         │             │  10.0.2.254         │        │
│  └────────┬────────────┘             └────────┬────────────┘        │
│           │                                   │                     │
│           └──────────── 物理网络 ──────────────┘                     │
└──────────────────────────────────────────────────────────────────────┘
```

#### 实现功能

程序依次执行以下步骤来构建完整的网络拓扑：

| 步骤 | 函数 | 说明 |
|------|------|------|
| 1 | `SetupNetNamespace()` | 创建网络命名空间 `ns0` |
| 2 | `SetupBridge()` | 创建 Linux 桥接设备 `bridge0` 并配置 IP |
| 3 | `SetupVEthPeer()` | 创建虚拟以太网对，连接命名空间与桥接 |
| 4 | `SetupNsDefaultRoute()` | 在命名空间中设置默认路由 |
| 5 | `SetupIPTables()` | 配置 NAT（MASQUERADE）规则实现外网访问 |
| 6 | `SetupRouteNs2Ns()` | 配置跨主机命名空间间的路由 |

#### 运行方法（需要 root 权限）

```bash
# 主机 A 上执行
IS_HOST_A=1 sudo -E go run pkg/my_bridge_net/main.go

# 主机 B 上执行
sudo go run pkg/my_bridge_net/main.go

# 测试连通性（在主机 A 上）
sudo ip netns exec ns0 ping 10.0.1.254    # ping 本机桥接
sudo ip netns exec ns0 ping 192.168.0.2   # ping 主机 B
sudo ip netns exec ns0 ping 10.0.2.1      # ping 主机 B 的命名空间
```

**核心知识点**: 网络命名空间、Linux 桥接、veth pair、NAT/MASQUERADE、路由表

> 详细文档请参阅 [docs/README_网络命名空间.md](docs/README_网络命名空间.md)

---

### 5. 工具库

**位置**: `util/`

提供 TCP 头部操作相关的核心工具函数。

#### 数据结构

```go
type TCPHeader struct {
    Source      uint16   // 源端口号
    Destination uint16   // 目标端口号
    SeqNum      uint32   // 序列号
    AckNum      uint32   // 确认号
    DataOffset  uint8    // 数据偏移（头部长度，单位：32位字）
    Reserved    uint8    // 保留位（3位）
    ECN         uint8    // 显式拥塞通知（3位）
    Ctrl        uint8    // 控制标志位（6位：URG/ACK/PSH/RST/SYN/FIN）
    Window      uint16   // 窗口大小
    Checksum    uint16   // 校验和
    Urgent      uint16   // 紧急指针
    Options     []TCPOption  // 可选项
}
```

#### TCP 控制标志位

| 常量 | 值 | 二进制 | 说明 |
|------|-----|--------|------|
| `FIN` | 1 | `000001` | 释放连接 |
| `SYN` | 2 | `000010` | 发起连接 |
| `RST` | 4 | `000100` | 重置连接 |
| `PSH` | 8 | `001000` | 推送数据 |
| `ACK` | 16 | `010000` | 确认有效 |
| `URG` | 32 | `100000` | 紧急指针有效 |

#### 核心函数

| 函数 | 说明 |
|------|------|
| `NewTCPHeader(data []byte) *TCPHeader` | 解析原始字节为 TCP 头部结构（仅处理目标端口为 888 的数据包） |
| `(tcp *TCPHeader) Marshal() []byte` | 将 TCP 头部结构序列化为字节切片（最小 20 字节） |
| `Csum(data []byte, srcip, dstip [4]byte) uint16` | 计算 TCP 校验和（包含伪头部） |
| `To4byte(addr string) [4]byte` | 将点分十进制 IP 地址转换为 4 字节数组 |

> 详细文档请参阅 [docs/README_工具库.md](docs/README_工具库.md)

---

## 学习路线

建议按照以下顺序学习本项目：

```
第1步：阅读 TCP 头部参考文档
  └── docs/tcp head.md
       │
第2步：标准 TCP 编程（入门）
  └── cmd/tcp/service.go + client.go
       │
第3步：理解 TCP 头部操作工具
  └── util/tcp_util.go
       │
第4步：自定义 TCP 头部构造（中级）
  └── cmd/socket/my_client.go + my_service.go
       │
第5步：原始套接字数据包捕获（高级）
  └── pkg/tcp/my_socket.go
       │
第6步：原始 IP 数据包操作（高级）
  └── cmd/sock_raw/service.go + client.go
       │
第7步：容器网络原理（高级）
  └── pkg/my_bridge_net/main.go
```

| 阶段 | 模块 | 难度 | 核心概念 |
|------|------|------|---------|
| 入门 | `cmd/tcp/` | 初级 | TCP 连接、并发处理、缓冲 I/O |
| 进阶 | `util/` + `cmd/socket/` | 中级 | TCP 头部格式、字节序、校验和 |
| 高级 | `pkg/tcp/` + `cmd/sock_raw/` | 高级 | 原始套接字、IP 层数据包操作 |
| 专家 | `pkg/my_bridge_net/` | 高级 | 网络命名空间、桥接、NAT、容器网络 |

---

## 依赖说明

### 主模块 (`go.mod`)

| 依赖 | 版本 | 说明 |
|------|------|------|
| `golang.org/x/net` | v0.27.0 | ICMP 解析、IPv4 工具 |
| `golang.org/x/sys` | v0.22.0 | 系统调用支持（间接依赖） |

### 网络命名空间模块 (`pkg/my_bridge_net/go.mod`)

| 依赖 | 版本 | 说明 |
|------|------|------|
| `github.com/vishvananda/netlink` | v1.3.0 | Linux 网络设备配置 |
| `github.com/vishvananda/netns` | v0.0.4 | 网络命名空间管理 |
| `github.com/coreos/go-iptables` | v0.8.0 | IPTables 规则操作 |

---

## 注意事项

1. **权限要求**：原始套接字和网络命名空间操作需要 root 权限，请使用 `sudo` 运行相关程序
2. **操作系统**：`cmd/sock_raw/`、`pkg/tcp/`、`pkg/my_bridge_net/` 仅支持 Linux 系统
3. **端口冲突**：默认使用端口 888，请确保该端口未被占用
4. **网络配置**：`pkg/my_bridge_net/` 中的 IP 地址需根据实际环境修改 `HostA` 和 `HostB` 变量
5. **仅限 IPv4**：`util.To4byte` 函数仅支持 IPv4 地址，不支持 IPv6

---

## 许可证

本项目仅用于学习和教育目的。

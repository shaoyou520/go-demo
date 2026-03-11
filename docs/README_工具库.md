# 工具库 API 文档

本文档详细介绍 `util/` 目录下的 TCP 头部操作工具库。

## 概述

`util` 包提供了 TCP 协议头部的解析、构造和校验和计算等核心功能，是自定义 TCP 头部构造和原始套接字监听模块的基础依赖。

## 文件说明

| 文件 | 说明 |
|------|------|
| `util/tcp_util.go` | TCP 头部结构体定义、解析、序列化和校验和计算 |
| `util/tcp_util_test.go` | 单元测试 |

---

## 常量定义

### TCP 控制标志位

```go
const (
    FIN = 1   // 00 0001 - 释放连接
    SYN = 2   // 00 0010 - 发起连接
    RST = 4   // 00 0100 - 重置连接
    PSH = 8   // 00 1000 - 推送数据，接收方应尽快交给应用层
    ACK = 16  // 01 0000 - 确认号有效
    URG = 32  // 10 0000 - 紧急指针有效
)
```

控制标志位可以组合使用，例如：
- `SYN + ACK = 18`：连接确认（三次握手第二步）
- `FIN + ACK = 17`：连接释放确认

---

## 数据结构

### TCPHeader

```go
type TCPHeader struct {
    Source      uint16      // 源端口号（16位）
    Destination uint16      // 目标端口号（16位）
    SeqNum      uint32      // 序列号（32位）
    AckNum      uint32      // 确认号（32位）
    DataOffset  uint8       // 数据偏移/头部长度（4位，单位：32位字）
    Reserved    uint8       // 保留位（3位）
    ECN         uint8       // 显式拥塞通知（3位）
    Ctrl        uint8       // 控制标志位（6位）
    Window      uint16      // 窗口大小（16位）
    Checksum    uint16      // 校验和（16位）
    Urgent      uint16      // 紧急指针（16位）
    Options     []TCPOption // TCP 选项（可变长度）
}
```

**内存布局与网络字节序的映射**：

```
字节位置:  0-1      2-3      4-7      8-11     12-13        14-15   16-17   18-19
字段:    Source  Destination SeqNum  AckNum  Offset+Flags  Window  Chksum  Urgent
```

其中第 12-13 字节的 16 位按以下方式分割：

```
位:    15 14 13 12 | 11 10 9 | 8  7  6 | 5  4  3  2  1  0
字段:  DataOffset  | Reserved |   ECN   |      Ctrl
       (4位)        (3位)      (3位)      (6位)
```

### TCPOption

```go
type TCPOption struct {
    Kind   uint8   // 选项类型
    Length uint8   // 选项长度
    Data   []byte  // 选项数据
}
```

---

## 函数详解

### NewTCPHeader - 解析 TCP 头部

```go
func NewTCPHeader(data []byte) *TCPHeader
```

**功能**：将原始字节数据解析为 `TCPHeader` 结构体。

**参数**：
- `data []byte`：原始 TCP 头部字节数据

**返回值**：
- `*TCPHeader`：解析后的 TCP 头部结构指针
- 如果目标端口不是 888，返回 `nil`

**解析过程**：

```
原始字节: [1F 90 03 78 00 00 00 6F 00 00 00 16 50 02 AA AA 00 00 00 63]
           ─────  ─────  ───────────  ───────────  ────  ─────  ─────  ─────
           Source  Dest    SeqNum       AckNum     Mix   Window Chksum Urgent
           8080    888     111          22         ...   43690  0      99
```

**位操作解析 Mix 字段**：
```go
var mix uint16
tcp.DataOffset = byte(mix >> 12)    // 取高4位
tcp.Reserved = byte(mix >> 9 & 7)   // 取第9-11位
tcp.ECN = byte(mix >> 6 & 7)        // 取第6-8位
tcp.Ctrl = byte(mix & 0x3f)         // 取低6位 (0011 1111)
```

**示例**：
```go
data := rawPacketBytes  // 原始 TCP 数据包
header := util.NewTCPHeader(data)
if header != nil {
    fmt.Printf("源端口: %d, 目标端口: %d\n", header.Source, header.Destination)
    fmt.Printf("序列号: %d, 确认号: %d\n", header.SeqNum, header.AckNum)
    fmt.Printf("控制标志: %d\n", header.Ctrl)
}
```

---

### Marshal - 序列化 TCP 头部

```go
func (tcp *TCPHeader) Marshal() []byte
```

**功能**：将 `TCPHeader` 结构体序列化为网络字节序的字节切片。

**返回值**：
- `[]byte`：序列化后的字节切片，最小 20 字节

**序列化过程**：

```go
// 1. 按序写入各字段
binary.Write(buf, binary.BigEndian, tcp.Source)
binary.Write(buf, binary.BigEndian, tcp.Destination)
binary.Write(buf, binary.BigEndian, tcp.SeqNum)
binary.Write(buf, binary.BigEndian, tcp.AckNum)

// 2. 组合 DataOffset、Reserved、ECN、Ctrl 为 16 位
mix = uint16(tcp.DataOffset)<<12 |
      uint16(tcp.Reserved)<<9 |
      uint16(tcp.ECN)<<6 |
      uint16(tcp.Ctrl)

// 3. 写入剩余字段和选项
// 4. 如果不足 20 字节，补 0
```

**补零机制**：TCP 头部最小为 20 字节。如果序列化后的数据不足 20 字节（当 Options 为空时不会出现这种情况），会自动补零。

**示例**：
```go
tcp := &util.TCPHeader{
    Source:      17663,
    Destination: 888,
    SeqNum:      2,
    DataOffset:  5,
    Ctrl:        util.SYN,
    Window:      0xaaaa,
}
data := tcp.Marshal()
// data 为 20 字节的网络字节序字节切片
```

---

### Csum - 计算 TCP 校验和

```go
func Csum(data []byte, srcip, dstip [4]byte) uint16
```

**功能**：计算 TCP 校验和，包含 TCP 伪头部。

**参数**：
- `data []byte`：TCP 头部 + 数据的字节切片
- `srcip [4]byte`：源 IP 地址（4 字节）
- `dstip [4]byte`：目标 IP 地址（4 字节）

**返回值**：
- `uint16`：计算得到的校验和值

**计算过程**：

```
TCP 伪头部（12 字节）:
┌─────────────────────────────────────┐
│     源 IP 地址（4 字节）             │
│     目标 IP 地址（4 字节）           │
│  0  │ 协议号(6) │ TCP 长度（2 字节） │
└─────────────────────────────────────┘

校验和计算:
1. 拼接伪头部 + TCP 数据
2. 按 16 位为单位累加
3. 如果长度为奇数，最后一个字节单独处理
4. 将进位加回低 16 位
5. 取反码
```

**示例**：
```go
data := tcp.Marshal()
checksum := util.Csum(data, util.To4byte("127.0.0.1"), util.To4byte("127.0.0.1"))
tcp.Checksum = checksum
```

---

### To4byte - IP 地址转换

```go
func To4byte(addr string) [4]byte
```

**功能**：将点分十进制的 IPv4 地址字符串转换为 4 字节数组。

**参数**：
- `addr string`：IPv4 地址字符串，如 `"192.168.1.1"`

**返回值**：
- `[4]byte`：4 字节的 IP 地址数组

**示例**：
```go
ip := util.To4byte("192.168.1.1")
// ip = [4]byte{192, 168, 1, 1}
```

> **注意**：此函数仅支持 IPv4 地址，不支持 IPv6。如果传入非法格式会导致程序崩溃。

---

## 单元测试

### 运行测试

```bash
go test ./util/...
```

### 测试用例说明

| 测试函数 | 说明 |
|---------|------|
| `TestNewTCPHeader` | 测试 TCP 头部解析（注意：目标端口为 80，不等于 888，所以返回 nil） |
| `TestTCPHeader_Marshal` | 测试 TCP 头部序列化 |
| `TestCsum` | 测试校验和计算 |

---

## 使用示例

### 完整的构造和发送流程

```go
package main

import (
    "github.com/shaoyou520/go-demo/util"
    "net"
)

func main() {
    // 1. 构造 TCP 头部
    tcp := &util.TCPHeader{
        Source:      12345,
        Destination: 888,
        SeqNum:      1,
        AckNum:      0,
        DataOffset:  5,
        Ctrl:        util.SYN,   // SYN 标志
        Window:      65535,
    }

    // 2. 序列化
    data := tcp.Marshal()

    // 3. 计算校验和
    srcIP := util.To4byte("127.0.0.1")
    dstIP := util.To4byte("127.0.0.1")
    tcp.Checksum = util.Csum(data, srcIP, dstIP)

    // 4. 重新序列化（包含校验和）
    data = tcp.Marshal()

    // 5. 附加负载
    payload := []byte("Hello, TCP!")
    data = append(data, payload...)

    // 6. 发送
    conn, _ := net.Dial("ip4:tcp", "127.0.0.1")
    conn.Write(data)
}
```

### 解析接收到的数据包

```go
// 假设 buf 是从原始套接字接收到的数据
header := util.NewTCPHeader(buf[0:20])
if header == nil {
    // 不是目标端口 888 的数据包，忽略
    return
}

// 检查控制标志
if header.Ctrl & util.SYN != 0 {
    fmt.Println("收到 SYN 包")
}
if header.Ctrl & util.ACK != 0 {
    fmt.Println("收到 ACK 包")
}

// 提取数据负载
payload := buf[header.DataOffset*4 : n]
fmt.Println("负载内容:", string(payload))
```

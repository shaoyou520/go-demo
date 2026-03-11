# 网络命名空间与桥接详解

本文档详细介绍 `pkg/my_bridge_net/` 目录下的 Linux 网络命名空间和桥接网络实现。

## 概述

本模块实现了类似 Docker 容器网络的完整网络隔离环境。通过创建网络命名空间、虚拟以太网对（veth pair）、Linux 桥接设备和 NAT 规则，模拟了容器网络的核心架构。

**实现目标**：
- 命名空间内的进程可以访问宿主机网络
- 不同主机上的命名空间可以互相通信

## 文件说明

| 文件 | 说明 |
|------|------|
| `pkg/my_bridge_net/main.go` | 完整的网络拓扑构建器 |
| `pkg/my_bridge_net/go.mod` | 独立模块定义（独立依赖管理） |

---

## 实验环境

### 双主机拓扑

```
┌─────────────────────────────────────────────────────────────────────────┐
│                                                                         │
│  主机 A (192.168.0.3)                    主机 B (192.168.0.2)           │
│  ┌───────────────────────┐               ┌───────────────────────┐      │
│  │  ns0 (网络命名空间)    │               │  ns0 (网络命名空间)    │      │
│  │                       │               │                       │      │
│  │  nseth0: 10.0.1.1/24  │               │  nseth0: 10.0.2.1/24  │      │
│  └───────────┬───────────┘               └───────────┬───────────┘      │
│              │ veth pair                              │ veth pair        │
│  ┌───────────┴───────────┐               ┌───────────┴───────────┐      │
│  │  veth0                │               │  veth0                │      │
│  │  bridge0: 10.0.1.254  │               │  bridge0: 10.0.2.254  │      │
│  └───────────┬───────────┘               └───────────┬───────────┘      │
│              │                                        │                 │
│              └────────── 物理网络(192.168.0.0/24) ─────┘                 │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### IP 地址规划

| 设备 | 主机 A | 主机 B |
|------|--------|--------|
| 物理网卡（HOST） | 192.168.0.3/24 | 192.168.0.2/24 |
| 桥接设备（bridge0） | 10.0.1.254/24 | 10.0.2.254/24 |
| 命名空间网卡（nseth0） | 10.0.1.1/24 | 10.0.2.1/24 |

### 通信目标

```
10.0.1.1 (主机A ns0) ──→ 192.168.0.2 (主机B)      # 跨主机通信
10.0.1.1 (主机A ns0) ──→ 10.0.2.1 (主机B ns0)     # 跨主机命名空间通信
```

---

## 核心概念

### 1. 网络命名空间（Network Namespace）

网络命名空间提供了网络资源的隔离，每个命名空间拥有独立的：
- 网络接口（网卡）
- 路由表
- 防火墙规则（iptables）
- 套接字

Docker 容器的网络隔离就是基于网络命名空间实现的。

### 2. 虚拟以太网对（veth pair）

veth pair 是一对虚拟网络接口，数据从一端进入会从另一端出来，就像一根虚拟网线。

```
┌─────────────┐          ┌─────────────┐
│   veth0     │◄────────►│   nseth0    │
│ (宿主机侧)  │  虚拟网线  │ (命名空间侧) │
└─────────────┘          └─────────────┘
```

### 3. Linux 桥接（Bridge）

Linux 桥接是一个虚拟的二层交换机，可以将多个网络接口连接在一起：

```
              ┌──────────┐
nseth0 ◄────►│          │
              │ bridge0  │◄────► 物理网卡
nseth1 ◄────►│ (虚拟交换机)│
              └──────────┘
```

### 4. NAT（MASQUERADE）

MASQUERADE 是一种源地址转换（SNAT）规则，将命名空间内的私有地址转换为宿主机的地址，使命名空间内的进程可以访问外部网络。

---

## 函数详解

### `main()` - 入口函数

```go
func main() {
    if os.Getenv(EnvName) == "1" {
        IsHostA = true
    }
    ns := SetupNetNamespace()
    bridge := SetupBridge()
    SetupVEthPeer(bridge, ns)
    SetupNsDefaultRoute()
    SetupIPTables()
    SetupRouteNs2Ns()
}
```

根据环境变量 `IS_HOST_A` 判断当前是主机 A 还是主机 B，然后依次执行六个配置步骤。

---

### `SetupNetNamespace()` - 创建网络命名空间

**等价命令**：
```bash
ip netns del ns0       # 删除已存在的命名空间
ip netns add ns0       # 创建新的命名空间
```

**核心逻辑**：
1. 使用 `runtime.LockOSThread()` 锁定当前 goroutine 到 OS 线程
2. 保存原始命名空间引用
3. 如果 `ns0` 已存在，先删除
4. 使用 `ip netns add` 命令创建新的命名空间
5. 返回命名空间句柄

**为什么需要锁定线程？**
Go 的 goroutine 可能在不同的 OS 线程间迁移。网络命名空间是与 OS 线程关联的，如果 goroutine 在切换命名空间后被迁移到其他线程，会导致操作在错误的命名空间中执行。

---

### `SetupBridge()` - 创建桥接设备

**等价命令**：
```bash
ip link add bridge0 type bridge        # 创建桥接
ip link set bridge0 up                 # 启动桥接
ip addr add 10.0.1.254/24 dev bridge0  # 设置 IP（主机 A）
```

**核心逻辑**：
1. 检查 `bridge0` 是否存在，存在则删除
2. 创建新的桥接设备（MTU=1500）
3. 启动桥接设备
4. 根据主机角色设置 IP 地址

---

### `SetupVEthPeer()` - 创建虚拟以太网对

**等价命令**：
```bash
ip link add veth0 type veth peer name nseth0     # 创建 veth pair
ip link set nseth0 netns ns0                     # 将一端移入命名空间
ip link set veth0 master bridge0                 # 将另一端连接到桥接
ip link set veth0 up                             # 启动宿主机端
# 在 ns0 命名空间中：
ip addr add 10.0.1.1/24 dev nseth0              # 设置 IP
ip link set nseth0 up                            # 启动命名空间端
```

**核心逻辑**：
1. 清理已存在的 `veth0`
2. 创建 veth pair（`veth0` + `nseth0`）
3. 将 `nseth0` 移入 `ns0` 命名空间
4. 将 `veth0` 连接到 `bridge0`
5. 启动 `veth0`
6. 在命名空间内为 `nseth0` 设置 IP 并启动

---

### `SetupNsDefaultRoute()` - 设置默认路由

**等价命令**（在命名空间中执行）：
```bash
ip netns exec ns0 ip route add default via 10.0.1.254
```

**功能**：在命名空间中添加默认网关，指向 `bridge0` 的 IP 地址，使命名空间内的流量可以通过桥接转发。

---

### `SetupIPTables()` - 配置 NAT 规则

**等价命令**：
```bash
iptables -t nat -A POSTROUTING -s 10.0.1.0/24 -j MASQUERADE
```

**功能**：为来自命名空间网段的数据包配置源地址伪装（MASQUERADE），实现 NAT 功能。这样命名空间内的进程就可以访问外部网络了。

**工作原理**：
```
ns0 (10.0.1.1) ──→ bridge0 ──→ MASQUERADE ──→ 外部网络
                                (10.0.1.1 → 192.168.0.3)
```

---

### `SetupRouteNs2Ns()` - 配置跨主机路由

**等价命令**（在主机 A 上执行）：
```bash
ip route add 10.0.2.0/24 via 192.168.0.2
```

**功能**：添加静态路由，使一台主机能够将数据包转发到另一台主机的命名空间网段。

**路由示意**：
```
主机 A:  10.0.2.0/24 → via 192.168.0.2 (主机B)
主机 B:  10.0.1.0/24 → via 192.168.0.3 (主机A)
```

---

### `NsDo()` - 命名空间操作辅助函数

```go
func NsDo(doFunc DoFunc) error {
    runtime.LockOSThread()
    defer runtime.UnlockOSThread()
    // 保存原始命名空间
    originNs, _ := netns.Get()
    defer func() {
        netns.Set(originNs)  // 恢复原始命名空间
    }()
    // 切换到目标命名空间
    ns, _ := netns.GetFromName(NSName)
    netns.Set(ns)
    // 在目标命名空间中执行函数
    return doFunc(&originNs, &ns)
}
```

**功能**：在指定命名空间中安全执行操作，执行完自动恢复到原始命名空间。

---

## 运行方法

### 前提条件

- 两台 Linux 主机（或虚拟机），且通过网络互通
- 已安装 `ip` 和 `iptables` 工具
- root 权限

### 步骤

#### 1. 在主机 A 上运行

```bash
IS_HOST_A=1 sudo -E go run pkg/my_bridge_net/main.go
```

#### 2. 在主机 B 上运行

```bash
sudo go run pkg/my_bridge_net/main.go
```

#### 3. 测试连通性

```bash
# 在主机 A 上测试
sudo ip netns exec ns0 ping 10.0.1.254     # ping 本机桥接 ✓
sudo ip netns exec ns0 ping 192.168.0.2    # ping 主机 B   ✓
sudo ip netns exec ns0 ping 10.0.2.1       # ping 主机 B ns0 ✓

# 在主机 B 上测试
sudo ip netns exec ns0 ping 10.0.2.254     # ping 本机桥接 ✓
sudo ip netns exec ns0 ping 192.168.0.3    # ping 主机 A   ✓
sudo ip netns exec ns0 ping 10.0.1.1       # ping 主机 A ns0 ✓
```

---

## 数据包路径分析

### 命名空间访问外部网络

```
ns0 (10.0.1.1)
    │
    │ 1. 数据包从 nseth0 发出
    │
    ▼
veth pair
    │
    │ 2. 数据到达 veth0（宿主机侧）
    │
    ▼
bridge0 (10.0.1.254)
    │
    │ 3. 桥接转发到宿主机网络栈
    │
    ▼
iptables MASQUERADE
    │
    │ 4. 源地址从 10.0.1.1 转换为 192.168.0.3
    │
    ▼
物理网卡 (192.168.0.3)
    │
    │ 5. 数据包发送到目标主机
    │
    ▼
目标主机
```

### 跨主机命名空间通信

```
ns0@主机A (10.0.1.1)
    │
    ▼
bridge0@主机A (10.0.1.254)
    │
    ▼
路由表: 10.0.2.0/24 via 192.168.0.2
    │
    ▼
物理网络 → 主机B (192.168.0.2)
    │
    ▼
路由表: 10.0.2.0/24 → bridge0
    │
    ▼
bridge0@主机B (10.0.2.254)
    │
    ▼
ns0@主机B (10.0.2.1)
```

---

## 与 Docker 网络的对比

| 组件 | 本项目 | Docker |
|------|--------|--------|
| 网络命名空间 | `ns0` | 每个容器一个命名空间 |
| 桥接设备 | `bridge0` | `docker0` |
| 虚拟网卡对 | `veth0`/`nseth0` | `vethXXX`/`eth0` |
| NAT 规则 | MASQUERADE | MASQUERADE |
| IP 分配 | 手动配置 | IPAM 驱动自动分配 |
| DNS | 无 | 内置 DNS 服务器 |

---

## 依赖库说明

| 库 | 说明 |
|----|------|
| `github.com/vishvananda/netlink` | Go 语言的 netlink 库，用于操作 Linux 网络设备（桥接、路由、IP 地址等） |
| `github.com/vishvananda/netns` | 网络命名空间管理库，提供创建、切换、获取命名空间的功能 |
| `github.com/coreos/go-iptables` | IPTables 规则操作库，用于配置 NAT 和防火墙规则 |

---

## 常见问题

### 1. 为什么使用 `ip netns add` 而不是纯 Go 代码创建命名空间？

代码中的注释解释了原因：
> 由于程序上启动 netns 是需要附着于进程上的，所以这里直接使用 `ip netns` 来创建 net namespace

使用 `ip netns add` 创建的命名空间会持久化在 `/var/run/netns/` 目录下，不会随进程退出而消失。

### 2. 为什么需要 `runtime.LockOSThread()`？

Go 的 goroutine 调度器可能在任意时刻将 goroutine 迁移到不同的 OS 线程。而网络命名空间是线程级别的属性，如果 goroutine 在切换命名空间后被调度到其他线程，会导致在错误的命名空间中操作。`LockOSThread` 确保当前 goroutine 固定在同一个 OS 线程上。

### 3. 如何修改 IP 地址适配自己的环境？

修改 `main.go` 中的以下常量：

```go
const (
    HostA        = "192.168.0.3/24"    // 主机 A 的物理 IP
    HostB        = "192.168.0.2/24"    // 主机 B 的物理 IP
    // 以下通常不需要修改
    HostANS0     = "10.0.1.1/24"
    HostABridge0 = "10.0.1.254/24"
    HostBNS0     = "10.0.2.1/24"
    HostBBridge0 = "10.0.2.254/24"
)
```

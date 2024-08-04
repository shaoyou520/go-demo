package sock_raw

import (
	"golang.org/x/net/ipv4"
	"net"
	"syscall"
)

func main() {
	//第一个参数：
	//
	//syscall.AF_INET，表示服务器之间的网络通信
	//syscall.AF_UNIX表示同一台机器上的进程通信
	//syscall.AF_INET6表示以IPv6的方式进行服务器之间的网络通信

	//第二个参数
	//
	//syscall.SOCK_RAW，表示使用原始套接字，可以构建传输层的协议头部，启用IP_HDRINCL的话，IP层的协议头部也可以构造，就是上面区分的传输层socket和网络层socket。
	//syscall.SOCK_STREAM, 基于TCP的socket通信，应用层socket。
	//syscall.SOCK_DGRAM, 基于UDP的socket通信，应用层socket。

	//第三个参数
	//即ICMP章节提到的子协议号，操作系统内核发现接收到的IP header中的协议号与创建时填的协议号一样时，就交给上层处理。
	//
	//IPPROTO_TCP 接收TCP协议的数据
	//IPPROTO_IP 接收任何的IP数据包
	//IPPROTO_UDP 接收UDP协议的数据
	//IPPROTO_ICMP 接收ICMP协议的数据
	//IPPROTO_RAW 只能用来发送IP数据包，不能接收数据。

	fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP)
	addr := syscall.SockaddrInet4{
		Port: 0,
		Addr: [4]byte{127, 0, 0, 1},
	}
	head := ipv4.Header{
		Version:  4,
		Len:      20,
		TotalLen: 20, // 20 bytes for IP
		TTL:      64,
		Protocol: 6, // TCP
		Dst:      net.IPv4(172, 17, 0, 3),
		Src:      net.IPv4(172, 17, 0, 99),
	}
	payload, _ := head.Marshal()
	syscall.Sendto(fd, payload, 0, &addr)
}

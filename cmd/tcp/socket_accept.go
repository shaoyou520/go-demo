package main

import (
	"github.com/shaoyou520/go-demo/pkg/tcp"
	"net"
)

func main() {
	netaddr, _ := net.ResolveIPAddr("ip4", "127.0.0.1")
	//传输层socket
	tcp.NetSocketTCP(netaddr)

}

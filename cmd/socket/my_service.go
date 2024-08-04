package main

import (
	"fmt"
	"github.com/shaoyou520/go-demo/util"
	"net"
)

func main() {
	netaddr, _ := net.ResolveIPAddr("ip4", "127.0.0.1")
	conn, _ := net.ListenIP("ip4:tcp", netaddr)
	for {
		buf := make([]byte, 1480)
		n, addr, _ := conn.ReadFrom(buf)
		ycpheader := util.NewTCPHeader(buf[0:20])
		if ycpheader == nil {
			continue
		}
		fmt.Println(n, addr, ycpheader, string(buf[20:n]))
	}
}

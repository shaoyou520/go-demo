package main

import (
	"fmt"
	"golang.org/x/net/icmp"
	"net"
)

func main() {
	netaddr, _ := net.ResolveIPAddr("ip4", "127.0.0.1")
	conn, _ := net.ListenIP("ip4:icmp", netaddr)
	for {
		buf := make([]byte, 1024)
		n, addr, _ := conn.ReadFrom(buf)
		msg, _ := icmp.ParseMessage(1, buf[0:n])
		fmt.Println(n, addr, msg.Type, msg.Code, msg.Checksum)
	}

}

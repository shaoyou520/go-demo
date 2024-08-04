package ping

import (
	"fmt"
	"github.com/shaoyou520/go-demo/util"
	"golang.org/x/net/icmp"
	"net"
)

func NetSocketICMP(netaddr *net.IPAddr) {
	conn, _ := net.ListenIP("ip4:icmp", netaddr)
	for {
		buf := make([]byte, 1024)
		n, addr, _ := conn.ReadFrom(buf)
		msg, _ := icmp.ParseMessage(1, buf[0:n])
		fmt.Println(n, addr, msg.Type, msg.Code, msg.Checksum)
	}
}

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

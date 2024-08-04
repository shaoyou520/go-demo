package sock_raw

import (
	"fmt"
	"golang.org/x/net/ipv4"
	"net"
)

func main() {
	netaddr, _ := net.ResolveIPAddr("ip4", "127.0.0.1")
	conn, _ := net.ListenIP("ip4:tcp", netaddr)
	ipconn, _ := ipv4.NewRawConn(conn)
	for {
		buf := make([]byte, 1500)
		hdr, payload, controlMessage, _ := ipconn.ReadFrom(buf)
		fmt.Println("ipheader:", hdr, payload, controlMessage)
	}
}

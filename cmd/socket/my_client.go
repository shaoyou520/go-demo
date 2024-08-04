package main

import (
	"github.com/shaoyou520/go-demo/util"
	"net"
	"time"
)

func main() {
	local := "127.0.0.1"
	remote := "127.0.0.1"
	conn, _ := net.Dial("ip4:tcp", remote)
	head := util.TCPHeader{
		Source:      17663,
		Destination: 888,
		SeqNum:      2,
		AckNum:      0,
		DataOffset:  5,
		Reserved:    0,
		ECN:         0,
		Ctrl:        2,
		Window:      0xaaaa,
		Checksum:    0,
		Urgent:      99,
	}
	data := head.Marshal()
	head.Checksum = util.Csum(data, util.To4byte(local), util.To4byte(remote))
	data = head.Marshal()
	data = append(data, []byte("你好, 李四, 你在想什么呢")...)
	for {
		conn.Write(data)
		time.Sleep(10 * time.Second)
	}

}

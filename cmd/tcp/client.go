package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	con, err := net.Dial("tcp", "127.0.0.1:888")
	defer con.Close()
	if err != nil {
		fmt.Println("dial failed, err:", err)
		return
	}
	for {
		_, err := con.Write([]byte("hello, zhangsan")) // 建立连接
		if err != nil {
			fmt.Println("accept failed, err:", err)
			continue
		}
		buf := [512]byte{}
		n, err := con.Read(buf[:])
		if err != nil {
			fmt.Println("recv failed, err:", err)
			return
		}
		fmt.Println(string(buf[:n]))
		time.Sleep(5 * time.Second)
	}
}

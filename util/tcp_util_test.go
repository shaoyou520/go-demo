package util

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
)

func newTcpHead() []byte {
	h := TCPHeader{
		Source:      8080,
		Destination: 80,
		SeqNum:      111,
		AckNum:      22,
		DataOffset:  3,
		Reserved:    4,
		ECN:         1,
		Ctrl:        2,
		Window:      55,
		Checksum:    44,
		Urgent:      55,
	}
	buf := &bytes.Buffer{}
	err := binary.Write(buf, binary.BigEndian, &h)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestNewTCPHeader(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name string
		args args
		want *TCPHeader
	}{
		// TODO: Add test cases.
		{name: "test1", args: args{data: newTcpHead()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			head := NewTCPHeader(tt.args.data)
			fmt.Println(head)
		})
	}
}

func TestTCPHeader_Marshal(t *testing.T) {
	tcp := &TCPHeader{
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
	t.Run("", func(t *testing.T) {
		tcp.Marshal()
	})

}

func TestCsum(t *testing.T) {
	type args struct {
		data  []byte
		srcip [4]byte
		dstip [4]byte
	}
	local := "127.0.0.1"
	remote := "127.0.0.1"
	tests := []struct {
		name string
		args args
		want uint16
	}{
		{name: "test1", args: args{data: []byte("123"), srcip: To4byte(local), dstip: To4byte(remote)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Csum(tt.args.data, tt.args.srcip, tt.args.dstip)
		})
	}
}

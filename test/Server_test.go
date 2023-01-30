package test

import (
	"encoding/json"
	"log"
	"net"
	"strconv"
	"testing"
	"tinyrpc/codec"
	"tinyrpc/server"
)

func TestServer(t *testing.T) {
	lis, _ := net.Listen("tcp", ":5000")
	go server.Accept(lis)
	log.Println("server 启动")
	conn, _ := net.Dial("tcp", ":5000")
	defer func() { _ = conn.Close() }()

	c := codec.MakeGobCodecFunc(conn)
	conArgs := codec.DefaultConArgs
	_ = json.NewEncoder(conn).Encode(conArgs)

	headers := make([]*codec.Header, 100)
	for i := 0; i < 5; i += 1 {
		headers[i] = &codec.Header{
			ServiceMethod: "Test.Add",
			Num:           uint64(i),
		}
		_ = c.WriteHeader(*headers[i])
		_ = c.WriteBody("send request " + strconv.Itoa(i))
	}
	//time.Sleep(10 * time.Second)
	for i := range headers {
		if headers[i] == nil {
			break
		}
		var reply string
		_ = c.ReadHeader(headers[i])
		_ = c.ReadBody(&reply)
		log.Println("reply: ", reply)
	}
}

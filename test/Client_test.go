package test

import (
	"context"
	"log"
	"net"
	"sync"
	"testing"
	"time"
	"tinyrpc/client"
	"tinyrpc/server"
)

func TestClient(t *testing.T) {
	lis, _ := net.Listen("tcp", ":5000")
	go func() {
		server.Accept(lis)
	}()

	c, err := client.Dial("tcp", ":5000")
	if err != nil {
		log.Println("Dial失败")
		return
	}
	defer func() {
		_ = c.Close()
	}()

	var group sync.WaitGroup
	for i := 0; i < 3; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			var reply string
			ctx, _ := context.WithTimeout(context.Background(), time.Second)
			if err := c.Call(ctx, "Hello.World", "wwb", &reply); err != nil {
				log.Println("call error:", err)
				return
			}
			log.Println("recieve :", reply)
		}()

	}
	group.Wait()
}

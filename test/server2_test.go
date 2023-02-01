package test

import (
	"errors"
	"log"
	"net"
	"sync"
	"testing"
	"tinyrpc/client"
	"tinyrpc/server"
)

type TestAdd struct {
}

type Argv struct {
	A int
	B int
}
type Reply struct {
	C int
}

func (t *TestAdd) Add(argv *Argv, reply *Reply) error {
	reply.C = argv.A + argv.B
	return nil
}
func (t *TestAdd) ReturnError(argv Argv, reply *Reply) error {
	reply.C = 100
	return errors.New("test : return error")
}
func TestServer2(t *testing.T) {

	err := server.Register(&TestAdd{})
	if err != nil {
		log.Println("register error:", err)
		return
	}
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
		log.Println("--------")
		err := c.Close()
		log.Println("err", err)
	}()

	var group sync.WaitGroup
	for i := 0; i < 3; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			argv := &Argv{A: 1, B: i}
			var reply Reply
			if err := c.Call("TestAdd.Add", argv, &reply); err != nil {
				log.Println("call error:", err)
				return
			}
			log.Println("recieve :", reply)
		}(i)
	}
	group.Wait()
	log.Println("-------")
}

package server

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"reflect"
	"strconv"
	"sync"
	"tinyrpc/codec"
)

// Server RPC调用服务端
type Server struct{}

func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("server error: server accept error", err)
			return
		}
		go server.ServeConn(conn)
	}
}

// ServeConn serve Connection
func (server *Server) ServeConn(conn net.Conn) {
	defer func() {
		_ = conn.Close()
	}()
	var conArgs codec.ConArgs
	if err := json.NewDecoder(conn).Decode(&conArgs); err != nil {
		log.Println("server error:decode conArgs error: ", err)
		return
	}
	if conArgs.Protocol != "rpc" {
		log.Println("server error: protocol error: ", conArgs.Protocol)
		return
	}
	f := codec.MakeCodecFuncMap[conArgs.CodecType]
	c := f(conn) //创建codec 编/译码器

	send := new(sync.Mutex)
	group := new(sync.WaitGroup)
	for {
		request, err := server.ReadRequest(c)
		if err != nil {
			if err != io.EOF && request.header != nil {
				request.header.Error = err
				request.reply = reflect.ValueOf("error")
				server.SendResponse(c, request, send)
			}
			break
		}
		group.Add(1)
		go server.HandleRequest(c, request, send, group)
	}
	group.Wait()
}

func (server *Server) ReadRequest(c codec.Codec) (*Request, error) {
	var header codec.Header
	log.Println("------------server read request------------")
	if err := c.ReadHeader(&header); err != nil {
		if err != io.EOF { //EOF代表读完，不应该输出error
			log.Println("server error: read request header", err)
		}
		return nil, err
	}
	request := &Request{
		header: &header,
	}
	request.argv = reflect.New(reflect.TypeOf(""))
	if err := c.ReadBody(request.argv.Interface()); err != nil {
		log.Println("server error: read request argv", err)
		return nil, err
	}
	log.Println("server decode request successfully", request.header, request.argv.Elem())
	return request, nil
}
func (server *Server) HandleRequest(c codec.Codec, request *Request, send *sync.Mutex, group *sync.WaitGroup) {
	defer group.Done()
	log.Println("------------server handle request------------")

	replyString := "rpc response your num " + strconv.Itoa(int(request.header.Num))
	request.reply = reflect.ValueOf(replyString)
	server.SendResponse(c, request, send)
}
func (server *Server) SendResponse(c codec.Codec, request *Request, send *sync.Mutex) {
	send.Lock()
	defer send.Unlock()
	if err := c.WriteHeader(*request.header); err != nil {
		log.Println("server error:write header ", err)
	}
	if err := c.WriteBody(request.reply.Interface()); err != nil {
		log.Println("server error:write body ", err)
	}
}

var defaultServer *Server

func Accept(lis net.Listener) {
	defaultServer.Accept(lis)
}

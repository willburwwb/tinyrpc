package server

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"
	"tinyrpc/codec"
)

// Server RPC调用服务端
type Server struct {
	services sync.Map // 所有注册的服务，索引为name 值为实例
}
type Request struct {
	header  *codec.Header
	argv    reflect.Value
	reply   reflect.Value
	service *Service
	method  *serviceMethod
}

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
		go server.HandleRequest(c, request, send, group, time.Second)
	}
	group.Wait()
}
func (server *Server) Register(serviceValue interface{}) error {
	s := NewService(serviceValue)
	if _, loaded := server.services.LoadOrStore(s.name, s); loaded {
		log.Println("server error: service has been loaded", s.name)
		return errors.New("server error: service has been loaded" + s.name)
	}
	log.Println(s.name + "register successfully")
	return nil
}
func (server *Server) findServiceAndMethod(serviceMethod string) (s *Service, m *serviceMethod) {
	str := strings.Split(serviceMethod, ".")
	serviceName := str[0]
	methodName := str[1]
	log.Printf("find service %s method %s\n", serviceName, methodName)
	sv, ok := server.services.Load(serviceName)
	if !ok {
		log.Println("server error: can`t find service", serviceName)
		return
	}
	s = sv.(*Service)
	m = s.method[methodName]
	if m == nil {
		log.Println("server error: can`t find method", methodName)
		return
	}
	return
}
func (server *Server) ReadRequest(c codec.Codec) (*Request, error) {
	var header codec.Header
	log.Println("------------server read request------------")
	if err := c.ReadHeader(&header); err != nil {
		if err != io.EOF { //EOF代表读完，不应该输出error
			log.Println("server error: read request header", err, header.ServiceMethod, header.Num)
		}
		return nil, err
	}
	request := &Request{
		header: &header,
	}
	//request.argv = reflect.New(reflect.TypeOf(""))

	request.service, request.method = server.findServiceAndMethod(header.ServiceMethod)
	if request.service == nil || request.method == nil {
		return nil, errors.New("server error: can't init request")
	}

	request.argv = request.method.newArgv()
	request.reply = request.method.newReply()

	if err := c.ReadBody(request.argv.Interface()); err != nil {
		log.Println("server error: read request argv", err)
		return nil, err
	}
	log.Println("server decode request successfully", request.header, request.argv.Elem())
	return request, nil
}
func (server *Server) HandleRequest(c codec.Codec, request *Request, send *sync.Mutex, group *sync.WaitGroup, timeout time.Duration) {
	defer group.Done()
	log.Println("------------server handle request------------")

	//replyString := "rpc response your num " + strconv.Itoa(int(request.header.Num))
	//request.reply = reflect.ValueOf(replyString)
	called := make(chan struct{})
	go func() {
		err := request.service.call(request.method, request.argv, request.reply)
		if err != nil {
			request.header.Error = err
		}
		called <- struct{}{}
	}()
	select {
	case <-time.After(timeout):
		request.header.Error = errors.New("server error: handle request timeout")
	case <-called:
	}
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

var defaultServer = &Server{}

func Accept(lis net.Listener) {
	defaultServer.Accept(lis)
}
func Register(serviceValue interface{}) error {
	return defaultServer.Register(serviceValue)
}
func SendHeartbeat(registryAddr string, addr string) error {
	httpClient := &http.Client{}

	req, _ := http.NewRequest("POST", registryAddr, nil)
	req.Header.Set("server", addr)
	if _, err := httpClient.Do(req); err != nil {
		log.Println("server error: heart beat err:", err)
		return err
	}
	log.Printf("server %v send heartbeat\n", addr)
	return nil
}
func ToSendHeartbeat(registryAddr string, addr string, timeout time.Duration) error {
	t := time.NewTicker(timeout)
	var err error
	for err == nil {
		<-t.C
		err = SendHeartbeat(registryAddr, addr)
	}
	return nil
}

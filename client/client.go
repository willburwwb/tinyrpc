package client

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"sync"
	"tinyrpc/codec"
)

// Client rpc_client 负责接收和转发数据
type Client struct {
	num       uint64
	conArgs   codec.ConArgs
	codecc    codec.Codec
	send      sync.Mutex
	clientMux sync.Mutex
	callQueue map[uint64]*Call
	closing   bool
}

func (client *Client) Close() error {
	client.clientMux.Lock()
	defer client.clientMux.Unlock()

	client.closing = true
	//client.broadcastCall(errors.New("client has been closed"))
	return client.codecc.Close()
}

//对于call的操作，加锁保护
func (client *Client) addCall(call *Call) error {
	client.clientMux.Lock()
	defer client.clientMux.Unlock()

	if client.isClosing() {
		return errors.New("client error: client is closing")
	}
	call.Num = client.num
	client.callQueue[client.num] = call
	client.num++
	return nil
}

func (client *Client) findCall(num uint64) *Call {
	client.clientMux.Lock()
	defer client.clientMux.Unlock()
	return client.callQueue[num]
}

func (client *Client) removeCall(num uint64) error {
	client.clientMux.Lock()
	defer client.clientMux.Unlock()
	if client.isClosing() {
		return errors.New("client error: client is closing")
	}
	delete(client.callQueue, num)
	return nil
}
func (client *Client) broadcastCall(err error) {
	client.clientMux.Lock()
	defer client.clientMux.Unlock()
	client.send.Lock()
	defer client.send.Unlock()
	for _, call := range client.callQueue {
		call.Error = err
		call.done()
	}
	client.closing = true
}
func (client *Client) isClosing() bool {
	return client.closing
}
func NewClient(conn net.Conn) *Client {
	client := &Client{
		num:       1,
		conArgs:   *codec.DefaultConArgs,
		callQueue: make(map[uint64]*Call),
	}

	f := codec.MakeCodecFuncMap[client.conArgs.CodecType]
	client.codecc = f(conn)
	if client.codecc == nil {
		return nil
	}
	return client
}

// receive 开启协程 阻塞接受conn信息
func (client *Client) receive() {
	var err error
	for {
		if err != nil {
			log.Println("client error :", err)
			break
		}
		var header codec.Header
		// 读入 header 中出错 不必接着读入body
		if err = client.codecc.ReadHeader(&header); err != nil {
			continue
		}
		call := client.findCall(header.Num)
		if call != nil {
			_ = client.removeCall(header.Num)
		} else {
			log.Println("the call has been removed")
			continue
		}
		//header
		if header.Error != nil {
			err = errors.New("client error: read header error " + err.Error())
			call.Error = err
			call.done()
			continue
		}
		if err = client.codecc.ReadBody(call.Reply); err != nil {
			call.Error = errors.New("client error:reading body" + err.Error())
		}
		log.Println("client read reply")
		call.done()
	}
	if err != nil {
		client.broadcastCall(err)
	}
}

// Dial 用于建立rpc_client与server 的连接,通过返回的Client可以同/异步调用服务端注册的方法。
func Dial(network string, addr string) (*Client, error) {
	conn, err := net.Dial(network, addr)
	//defer func() { _ = conn.Close() }()注意这里不要随手close掉。。。。
	if err != nil {
		log.Println("client error:client connection error", err)
		return nil, err
	}
	// 先进行协议上的沟通，出于方便同一采用defaultConArgs
	err = json.NewEncoder(conn).Encode(codec.DefaultConArgs)
	log.Println("client send conArgs-------", codec.DefaultConArgs)
	if err != nil {
		return nil, err
	}

	client := NewClient(conn)
	if client == nil {
		log.Println("client error:new client failed")
		return nil, errors.New("new client failed")
	}
	go client.receive()
	return client, nil
}

func (client *Client) sendCall(call *Call) {
	client.send.Lock()
	defer client.send.Unlock()

	header := &codec.Header{
		ServiceMethod: call.ServerMethod,
		Num:           call.Num,
	}
	log.Println("client send header", header)
	if err := client.codecc.WriteHeader(*header); err != nil {
		_ = client.removeCall(call.Num)
		if call != nil {
			call.Error = err
			call.done()
		}
		return
	}
	if err := client.codecc.WriteBody(call.Argv); err != nil {
		_ = client.removeCall(call.Num)
		if call != nil {
			call.Error = err
			call.done()
		}
		return
	}
}

// Go invokes the function asynchronously. It returns the Call structure representing
// the invocation. The done channel will signal when the call is complete by returning
// the same Call object. If done is nil, Go will allocate a new channel.
// If non-nil, done must be buffered or Go will deliberately crash.
func (client *Client) Go(serviceMethod string, argv interface{}, reply interface{}, done chan *Call) *Call {
	if done == nil || cap(done) == 0 {
		done = make(chan *Call, 1)
	}
	call := NewCall(serviceMethod, argv, reply, done)
	if err := client.addCall(call); err != nil {
		call.Error = err
		call.done()
		return call
	}

	client.sendCall(call)
	return call
}
func (client *Client) Call(ctx context.Context, serviceMethod string, argv interface{}, reply interface{}) error {
	call := client.Go(serviceMethod, argv, reply, make(chan *Call, 1))
	//call = <-call.Done
	select {
	case <-ctx.Done():
		_ = client.removeCall(call.Num)
		return errors.New("rpc client: timeout")
	case call = <-call.Done:
		return call.Error
	}
}

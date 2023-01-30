/*
	codec 用于实现传输信息的序列化和反序列化
*/
package codec

import "io"

type Type string

// ConArgs 建立连接时互相确认的参数
type ConArgs struct {
	Protocol  string
	CodecType Type
}

// Header 调用的头部信息
type Header struct {
	Num           uint64 //请求序号
	ServiceMethod string //方法名称
	Error         error
}

// Codec 用于实现不同编解码器的接口
type Codec interface {
	ReadHeader(header *Header) error
	ReadBody(body interface{}) error
	WriteHeader(header Header) error
	WriteBody(body interface{}) error
	Close() error
}
type MakeCodecFunc func(conn io.ReadWriteCloser) Codec

var DefaultConArgs = &ConArgs{
	Protocol:  "rpc",
	CodecType: "gob",
}
var MakeCodecFuncMap map[Type]MakeCodecFunc

func init() {
	MakeCodecFuncMap = make(map[Type]MakeCodecFunc)
	MakeCodecFuncMap["gob"] = MakeGobCodecFunc //map中储存不同数据类型对应的构造函数，可水平拓展
}

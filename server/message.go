package server

import (
	"reflect"
	"tinyrpc/codec"
)

type Request struct {
	header *codec.Header
	argv   reflect.Value
	reply  reflect.Value
}

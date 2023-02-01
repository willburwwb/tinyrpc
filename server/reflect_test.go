package server

import (
	"errors"
	"log"
	"reflect"
	"testing"
)

type TestAdd struct {
}

type Argv struct {
	a int
	b int
}
type Reply struct {
	c int
}

func (t *TestAdd) Add(argv Argv, reply *Reply) error {
	reply.c = argv.a + argv.b
	return nil
}
func (t *TestAdd) ReturnError(argv Argv, reply *Reply) error {
	reply.c = 100
	return errors.New("test : return error")
}
func TestReflect(t *testing.T) {
	s := NewService(&TestAdd{})

	serviceMethodAdd := s.method["Add"]
	argv := serviceMethodAdd.newArgv()
	reply := serviceMethodAdd.newReply()

	a := Argv{a: 1, b: 5}
	argv.Set(reflect.ValueOf(a))

	err := s.call(serviceMethodAdd, argv, reply)
	log.Printf("reply %v error %v", reply, err)

	serviceMethodError := s.method["ReturnError"]

	err = s.call(serviceMethodError, argv, reply)
	log.Printf("---reply %v error %v", reply, err)
}

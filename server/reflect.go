package server

import (
	"log"
	"reflect"
)

// serviceMethod 记录每个服务的方法的类型
type serviceMethod struct {
	method    reflect.Method
	argvType  reflect.Type
	replyType reflect.Type
}

// 通过serviceMethod中Argv和Reply类型返回对应的reflect.Value
func (m *serviceMethod) newArgv() (argv reflect.Value) {
	if m.argvType.Kind() == reflect.Ptr {
		argv = reflect.New(m.argvType.Elem())
	} else {
		argv = reflect.New(m.argvType).Elem()
	}
	return
}
func (m *serviceMethod) newReply() (reply reflect.Value) {
	// 只能是指针类型
	reply = reflect.New(m.replyType.Elem())
	return
}

// Service 定义服务，需要被注册，
type Service struct {
	name         string
	serviceType  reflect.Type
	serviceValue reflect.Value
	method       map[string]*serviceMethod
}

// NewService 创建service实例，并且将service对应的方法注册到service中
func NewService(serviceValue interface{}) *Service {
	s := new(Service)
	s.serviceValue = reflect.ValueOf(serviceValue)
	s.serviceType = reflect.TypeOf(serviceValue)
	log.Printf("serviceType %v serviceValue %v\n", s.serviceType, s.serviceValue)
	s.name = s.serviceType.Elem().Name()
	s.method = make(map[string]*serviceMethod)
	for i := 0; i < s.serviceType.NumMethod(); i++ {
		serviceMethodType := s.serviceType.Method(i).Type

		if serviceMethodType.NumIn() != 3 || serviceMethodType.NumOut() != 1 {
			log.Printf("service %v method %v wrong format\n", s.serviceType.Name(), serviceMethodType.Name())
			continue
		}
		//得到service对应每个方法的reflect.Type
		s.method[s.serviceType.Method(i).Name] = &serviceMethod{
			argvType:  serviceMethodType.In(1),
			replyType: serviceMethodType.In(2),
			method:    s.serviceType.Method(i),
		}
		log.Printf("service %v register %v\n", s.name, s.serviceType.Method(i).Name)
	}
	return s
}
func (s *Service) call(m *serviceMethod, argv reflect.Value, reply reflect.Value) error {
	fc := m.method.Func
	errorValues := fc.Call([]reflect.Value{s.serviceValue, argv, reply})
	errorValue := errorValues[0].Interface()
	if errorValue != nil {
		return errorValue.(error)
	}
	return nil
}

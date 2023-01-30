package client

type Call struct {
	Num          uint64
	ServerMethod string
	Argv         interface{}
	Reply        interface{}
	Error        error
	Done         chan *Call
}

// done 利用channel异步通知当前调用结束
func (call *Call) done() {
	call.Done <- call
}

func NewCall(serviceMethod string, argv interface{}, reply interface{}, done chan *Call) *Call {
	return &Call{
		ServerMethod: serviceMethod,
		Argv:         argv,
		Reply:        reply,
		Done:         done,
	}
}

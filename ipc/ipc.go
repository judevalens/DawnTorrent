package ipc

import (
	"fmt"
	zmq "github.com/pebbe/zmq4"
)

func InitZeroMQ()  {
	context, _ := zmq.NewContext()

	fmt.Printf("%v",context)
}
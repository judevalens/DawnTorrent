package ipc

import (
	"fmt"
	zmq "github.com/pebbe/zmq4"
	"log"
)


func InitZMQ(){
	context, _ := zmq.NewContext()

	server, _ := zmq.NewSocket(zmq.REP)

	errIpcServer := server.Bind("tcp://*:5555")

	if errIpcServer != nil{
		log.Fatal(errIpcServer)
	}
	fmt.Printf("\n context zmq %v\n", context)


	for {
		msg, _ := server.Recv(zmq.SNDMORE)
		_, _ = server.Send("msg "+msg, zmq.DONTWAIT)
		log.Println(msg)
		//os.Exit(222)
	}
	//os.Exit(222)
}


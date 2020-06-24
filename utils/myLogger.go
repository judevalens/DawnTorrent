package utils

import (
	"bytes"
	"log"
)

type MyLogger struct {
	logger *log.Logger
	on bool
}


func NewMyLogger(flag int, prefix string) *MyLogger{
	l := new(MyLogger)
	buffer := bytes.NewBuffer(make([]byte,0))
	l.logger = log.New(buffer,prefix,flag)

	return l
}


func (l *MyLogger) log(s string){

}

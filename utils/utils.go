package utils

import (
	"fmt"
)

const DEBUG = true


func Debugln(st string){
	if DEBUG {
		fmt.Println(st)
	}
}
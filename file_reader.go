package main

import (
	"io"
	"io/ioutil"
	"log"

)

type Dict struct {
	values map[string]string
	list 	map[string]string
	embedDict  map[string]Dict
}

func readFile(path string)  []byte{
	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	//fmt.Printf("File contents: %s", file)

	return file
}

func decode(reader io.Reader){

}


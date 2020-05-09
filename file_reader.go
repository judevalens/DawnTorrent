package main

import (
	"fmt"
	"io/ioutil"
	"log"
)

func readFile(path string)  []byte{
	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("File contents: %s", file)

	return file
}


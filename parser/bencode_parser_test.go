package parser

import (
	"bytes"
	"fmt"
	"testing"
)

func TestUnmarshallFromArray(t *testing.T)	{


	var _ = []struct{
		input string
		output	*Dict
	}{
		{"/home/jude/GolandProjects/DawnTorrent/files/ubuntu-20.04-desktop-amd64.iso.torrent",nil},
	}
	//TODO write more extensive unit test for this module

}

func TestReadFile(t *testing.T) {
	var readFileTest1 = []struct {
		path string
		output []byte
		errNil bool
	}{
		{"/home/jude/GolandProjects/DawnTorrent/files/readingFile1Test.txt", []byte("hello world"),true},
		{"/home/jude/GolandProjects/DawnTorrent/files/readingFile2Test.txt", []byte("hello world 2"),true},
		{"/home/jude/GolandProjects/DawnTorrent/files/FAIL.txt", nil,false},
	}

	for i,	cases := range readFileTest1{
		testName := fmt.Sprintf("read file test %v",i)
			t.Run(testName, func(t *testing.T) {
				readFileOutput,readFileErr := ReadFile(cases.path)
				ans := bytes.Compare(readFileOutput , cases.output)

				if ans != 0 && readFileErr == nil == cases.errNil{
					t.Errorf("got %v, want %v",string(readFileOutput),string(cases.output))
				}
			})
	}


}
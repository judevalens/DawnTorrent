package parser

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUnmarshallFromArray(t *testing.T)	{


	var _ = []struct{
		input string
		output	*BMap
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
		{"../files/readingFile1Test.txt", []byte("hello world"),true},
		{"../files/readingFile2Test.txt", []byte("hello world 12s"),true},
	}

	for i,	cases := range readFileTest1{
		testName := fmt.Sprintf("read file test %v",i)
			t.Run(testName, func(t *testing.T) {
				readFileOutput,readFileErr := ReadFile(cases.path)

				if readFileErr != nil{
					assert.Fail(t, readFileErr.Error())
					return
				}

				assert.Equal(t,string(cases.output),string(readFileOutput))
			})
	}


}
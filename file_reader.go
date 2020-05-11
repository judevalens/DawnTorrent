package main

import (
	"container/list"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
)

const (
	Dictionary = "d"
	List       = "l"
	Number     = "i"
	Closer     = "e"
)

var openDel list.List
var closing list.List

type key struct {
	key   string
	dType string
}

type dictList struct {
	lString []string
	lInt    []int
	lDict   []*Dict
	lList   []*dictList
	value   string
}

type Dict struct {
	dataList []key
	mString  map[string]string
	mInt     map[string]int
	mDict    map[string]*Dict
	mList    map[string]*dictList
}

type Result struct {
	dict  *Dict
	dList *dictList
	position int
	field 	byte
}

type Delimiter struct {
	delType       string
	contentLength []int
	content       string
	number        int
	position      int
}

func readFile(path string) []byte {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	return file
}

func Decode(file []byte, offset int) {
	//dict := Dict{}

	contentLength := 0
	//fileLength := len(file)
	fmt.Printf("FILE LENGTH")

	for offset < len(file) {
		fmt.Printf("FILE LENGTH %v OFFSET %v\n", len(file), offset)

		s, delimiter := getInfo(file, offset)
		totalBytesRead := len(s)

		fmt.Printf("\n%s\n", s)
		dataType := s[:delimiter]
		println("Delimiter %v", delimiter)
		contentLength, _ = strconv.Atoi(strings.Join(s[delimiter:totalBytesRead-1], ""))
		fmt.Printf("type %s length %v\n", dataType, contentLength)

		for i, v := range dataType {
			var del Delimiter

			if v == Dictionary || v == List || v == Number {

				if i == len(dataType)-1 {
					//content := file[offset:contentLength]
					del.delType = v

				} else {

					del = Delimiter{delType: v}
				}

				openDel.PushBack(del)
				closing.PushFront(del)
			}

		}
		offset += totalBytesRead + contentLength

	}

	println("-----------------------------------\n")

	del := openDel.Front()

	for del != nil {
		fmt.Printf("%v\n", del.Value.(Delimiter).delType)

		del = del.Next()
	}

	del = closing.Front()

	for del != nil {
		fmt.Printf("PREV %v\n", del.Value.(Delimiter).delType)
		del = del.Next()
	}

}

func getInfo(file []byte, offset int) ([]string, int) {
	var ch string
	seq := make([]string, 0)
	delimiter := 0
	isNumber := false
	for ch != ":" && offset < len(file) {

		ch = string(file[offset])
		seq = append(seq, ch)

		if strings.Contains("dlie", ch) && ch != "" {
			println("del %v", ch)
			delimiter++

			if ch == "i" {
				isNumber = true
			}
			if ch == "e" {
				isNumber = false
			}
		} else {
			if isNumber {
				delimiter++
			}
		}

		offset++
	}

	// fixed a small issue
	// there's probably a more elegant way to solve it :)
	if offset >= len(file) {
		seq = append(seq, ":")
	}
	return seq, delimiter
}

func getField(file []byte, pos int, dtype byte) ([]string, int, byte) {
	var ch string
	var field byte

	seq := make([]string, 0)
	_ = 0
	_ = false

	fmt.Printf("dtype %v\n", string(dtype))
	counter := 0
	for counter < 2 {
		ch = string(file[pos])
		lengthS := ""

		isNumber := false
		var isEnd bool

		if ch == "e" {
			isEnd = true
		}else{
			isEnd = false
		}

		for ch != "d" && ch != "l" && ch != ":" && !isEnd {
			println("char " + ch)

			lengthS += ch
			pos++
			ch = string(file[pos])

			if ch == "i" {
				isNumber = true
			}

			if ch == "e" {
				if isNumber{
					isNumber = false
				}else{
					isEnd = true
				}
			}

		}

		if file[pos] == 'd' || file[pos] == 'l' || file[pos] == 'e' {
			field = file[pos]
		}

		if ch == ":" {
			length, _ := strconv.Atoi(lengthS)
			pos++
			println("LENGTH " + lengthS)
			content := string(file[pos : pos+length])
			println("content " + content)

			seq = append(seq, content)
			pos = pos + length
		}
		counter++
	}

	return seq, pos, field
}

func parse(dict *Dict, file []byte, pos int) Result {

	field := file[pos]
	position := pos
	position++

	stack := 0

	if field == 'd' {
		dict.dataList = make([]key, 0)
		dict.mString = make(map[string]string, 0)
		dict.mList = make(map[string]*dictList)
		dict.mDict = make(map[string]*Dict)

		for field != 'e' {
			data, p, f := getField(file, position, field)
			position = p
			field = f
			if field == 0 || field == 'e' {
				key := key{
					key:   data[0],
					dType: "d",
				}
				dict.dataList = append(dict.dataList, key)

				dict.mString[data[0]] = data[1]
				println("adding string  dict")
				println("Field " + string(field))
			} else if field == 'l' {
				key := key{
					key:   data[0],
					dType: "l",
				}
				result := parse(dict, file, position)


				dict.dataList = append(dict.dataList, key)
				dict.mList[data[0]] = result.dList
				position = result.position
				println("adding list to dict")
				println("Field " + string(field))
			} else if field == 'd' {
				key := key{
					key:   data[0],
					dType: "d",
				}
				dict.dataList = append(dict.dataList, key)

				result := parse(dict, file, position)
				position = result.position



				dict.mDict[data[0]] = result.dict
				println("adding dict to dict")
				println("Field " + string(field))
			}
			_ = position

		}

		return Result{dict: dict,dList: new(dictList),position: position,field: field}

	} else if field == 'l' {

		dList := new(dictList)
		dList.lString = make([]string, 0)
		dList.lList = make([]*dictList, 0)
		dList.lDict = make([]*Dict, 0)
		for field != 'e' {
			data, p, f := getField(file, position, field)
			position = p
			field = f
			if field == 'e' {
				dList.lString = append(dList.lString, data[0])
				println("adding string to list")
				println("Field " + string(field))
			} else if field == 'l' {

				result := parse(dict, file, position)
				dList.lList = append(dList.lList, result.dList)
				position = result.position
				position++
				println("adding list to list")
				println("Field " + string(field))
			} else if field == 'd' {
				result := parse(dict, file, position)
				dList.lDict = append(dList.lDict, result.dict)
				position = result.position
				println("adding dict to list")
				println("Field " + string(field))
			}

		}
		_ = position

		return Result{dict: new(Dict), dList: dList, position: position,field: field}
	}

	return Result{}
}

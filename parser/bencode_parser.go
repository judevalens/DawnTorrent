package parser

import (
	"io/ioutil"
	"strconv"
)

const (
	Bdictionnary = 'd'
	Blist        = 'l'
	Bnumber      = 'i'
	Bstring      = 's'
	bEnd         = 'e'
)



type containerInformation struct {
	key                   string
	dType                 string
	StartingPosition      int
	EndingPosition        int
	InnerStartingPosition int
	InnerEndingPosition   int
	Index                 int
}

type BList struct {
	DataList     []*containerInformation
	Strings      []string
	BMaps        []*BMap
	BLists       []*BList
	value        string
	KeyInfo      *containerInformation
	OriginalFile []byte
}

type BMap struct {
	DataList []*containerInformation
	Strings  map[string]string
	BMaps    map[string]*BMap
	BLists   map[string]*BList
	KeyInfo  *containerInformation
}

type Result struct {
	dict     *BMap
	dList    *BList
	position int
	field    byte
}

type Delimiter struct {
	delType       string
	contentLength []int
	content       string
	number        int
	position      int
}

func Unmarshall(path string) (*BMap,error) {

	file, err := ReadFile(path)
	if err != nil{
		return nil, err
	}
	return UnmarshallFromArray(file)
}

func UnmarshallFromArray(file []byte) (*BMap,error) {

	dict := new(BMap)
	containerInfo := new(containerInformation)
	containerInfo.key = "origin"
	containerInfo.dType = "d"
	containerInfo.StartingPosition = 0
	r := Parse(dict, file, 0)
	containerInfo.InnerEndingPosition = r.position

	dict.KeyInfo = containerInfo

	return dict,nil

}

func ReadFile(path string) ([]byte,error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
	//	log.Fatal(err)
	}
	return file,err
}
func getField(file []byte, pos int, dtype byte) ([]string, int, byte) {
	var ch string
	var field byte

	seq := make([]string, 0)
	_ = 0
	_ = false
//	fmt.Printf("dtype %v\n", string(dtype))

	counter := 0
	for counter < 2 {
		ch = string(file[pos])
		lengthS := ""

		var isNumber bool = false
		var isEnd bool = false

		if ch == "e" {
			isEnd = true
		}

		numberS := ""

		for ch != "d" && ch != "l" && ch != ":" && !isEnd {
		//	println("char in at pos " + strconv.Itoa(pos) + " " + ch)

			if ch == "i" {
				isNumber = true
			}

			lengthS += ch
			pos++
			ch = string(file[pos])

			if ch == "e" {
				isEnd = true
				isNumber = true
			}

			if isNumber {
				_, err := strconv.Atoi(ch)

				if err == nil {
					numberS += ch
				}
			}

		}

		if file[pos] == 'd' || file[pos] == 'l' || file[pos] == 'e' {
			field = file[pos]
		}

		if ch == ":" {
			length, _ := strconv.Atoi(lengthS)
			pos++
		//	println("LENGTH " + lengthS)
			content := string(file[pos : pos+length])
		//	println("content " + content)

			seq = append(seq, content)
			pos = pos + length
		} else if isNumber {
	//		println("NUMBER " + numberS)

			seq = append(seq, numberS)
			pos++
		}
		counter++
	}

	return seq, pos, field
}

func Parse(dict *BMap, file []byte, pos int) Result {

	// determine which container it is (map or list)
	print(len(file))
	field := file[pos]
	position := pos
	position++

	// EOL -> end of a container
	// used to know when to backtrack, when there is no more element to add to container
	EOL := false

	if field == 'd' {

		dict.DataList = make([]*containerInformation, 0)
		dict.Strings = make(map[string]string, 0)
		dict.BLists = make(map[string]*BList)
		dict.BMaps = make(map[string]*BMap)

		for !EOL {

			containerInfo := new(containerInformation)

			//Store the stating position of this container in the bencoded file
			containerInfo.StartingPosition = position

			/*
				getField reads the next field if its a simple child (String) we added it to the current dictionary
				if it's it a container, we create and process the content of that content of that container first then add it to the this current container
			*/
			data, p, f := getField(file, position, field)

			position = p
			field = f

			containerInfo.key = data[0]
			//store the starting of the content of the container
			containerInfo.InnerStartingPosition = position

			//field = 0 -> normal field (a string), field = d -> innerDictionary, field = l -> innerList

			if field == 0 || field == 'e' {
				containerInfo.dType = "s"
				dict.DataList = append(dict.DataList, containerInfo)
				dict.Strings[data[0]] = data[1]

			//	println("key " + data[0])
			///	println("value " + data[1])
			//	println("adding string  dict")
			//	println("Field " + string(field))
			} else if field == 'l' {

				result := Parse(dict, file, position)
				dict.BLists[data[0]] = result.dList
				containerInfo.dType = "l"
				result.dList.KeyInfo = containerInfo
				dict.DataList = append(dict.DataList, containerInfo)
				position = result.position
				position++
			//	println("key " + data[0])
			//	println("adding list to dict")
			//	println("Field " + string(field))
			} else if field == 'd' {

				dict.DataList = append(dict.DataList, containerInfo)
				innerDict := new(BMap)
				result := Parse(innerDict, file, position)
				containerInfo.dType = "d"

				result.dict.KeyInfo = containerInfo
				position = result.position
				position++
				dict.BMaps[data[0]] = result.dict
				dict.KeyInfo = containerInfo

			//	println("position " + strconv.Itoa(position))
			///	println("adding dict to dict")
			/////	println("Field " + string(field))

			}
			if file[position] == 'e' {
				EOL = true
			}
			containerInfo.EndingPosition = position
			containerInfo.InnerEndingPosition = position - 1
			_ = position
		}

		return Result{dict: dict, dList: new(BList), position: position, field: field}

	} else if field == 'l' {

		dList := new(BList)
		dList.Strings = make([]string, 0)
		dList.BLists = make([]*BList, 0)
		dList.BMaps = make([]*BMap, 0)
		for !EOL {
			containerInfo := new(containerInformation)
			containerInfo.StartingPosition = position
			containerInfo.Index = len(dList.DataList)

			data, p, f := getField(file, position, field)
			position = p
			field = f

			containerInfo.InnerStartingPosition = position
			if field == 'e' {
				containerInfo.dType = "s"
				dList.DataList = append(dList.DataList,containerInfo)
				dList.Strings = append(dList.Strings, data[0])
			//	println("adding string to list")
			//	println("Field " + string(field))

			} else if field == 'l' {
				containerInfo.dType = "l"
				containerInfo.key = strconv.Itoa(len(dList.BLists))

				dList.DataList = append(dList.DataList,containerInfo)

				result := Parse(dict, file, position)
				dList.KeyInfo = containerInfo
				dList.BLists = append(dList.BLists, result.dList)
				position = result.position
				position++
			//	println("adding list to list")
			//	println("Field " + string(field))
			} else if field == 'd' {
				containerInfo.dType = "d"
				dList.DataList = append(dList.DataList,containerInfo)

				containerInfo.key = strconv.Itoa(len(dList.BMaps))
				innerDict := new(BMap)
				result := Parse(innerDict, file, position)
				dList.KeyInfo = containerInfo
				dList.BMaps = append(dList.BMaps, result.dict)
				position = result.position
				position++
			//	println("adding dict to list")
			//	println("Field " + string(field))
			}
			if file[position] == 'e' {
				EOL = true
			}

			containerInfo.EndingPosition = position
			containerInfo.InnerEndingPosition = position - 1
			_ = position
		}

		return Result{dict: new(BMap), dList: dList, position: position, field: field}
	}

	return Result{}
}

func ToBencode(container interface{}) string{
	var containerS string
	switch container := container.(type) {
	case *BMap:
		//println("ITS A DICT")
		containerS += "d"
		for _, info := range container.DataList {
			keyLen := len(info.key)
			containerS += strconv.Itoa(keyLen) + ":" + info.key
			switch info.dType {
			case "s":
				content := container.Strings[info.key]
				contentLen := len(content)

				_,isNumber := strconv.Atoi(content)

				// if the content is number with format it to BENCODE
				if isNumber == nil{
					content  = "i"+content+"e"
				}else{
					content =  strconv.Itoa(contentLen)+":" + content
				}

				containerS += content
			case "l":
				result := ToBencode(container.BLists[info.key])
				containerS += result
			case "d":
				result := ToBencode(container.BMaps[info.key])
				containerS += result
			}

		}
		containerS += "e"
		return containerS

	case *BList:
		containerS += "l"
		for _,info := range container.DataList{
			switch info.dType {
			case "s":
				content := container.Strings[info.Index]
				contentLen := len(content)
				containerS += strconv.Itoa(contentLen)+ ":"+content
			case "d":
				result := ToBencode(container.BMaps[info.Index])
				containerS += result
			case "l":
				result := ToBencode(container.BLists[info.Index])
				containerS += result

			}
		}

		containerS += "e"

		return containerS
	}
return ""
}

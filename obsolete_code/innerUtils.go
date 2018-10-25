package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

func loadConfigContent(isStdin bool, base64Data string, filename string, isForce bool, isOffset bool) (content string, isBase64 bool, err error) {
	content = ""
	isBase64 = false

	if isStdin {
		//其实,可以从标准输入中读取整个配置文件的,
		//因为文件的内容可能有多行,不太好读取,
		//所以仅支持从标准输入中读取配置文件的base64编码后的内容.
		if _, err = fmt.Scanln(&content); err != nil {
			content = ""
		}
		isBase64 = true
		return
	}

	if 0 < len(base64Data) {
		content = base64Data
		isBase64 = true
		return
	}

	if 0 < len(filename) {
		if isOffset && !path.IsAbs(filename) {
			filename = path.Join(os.Args[0][:strings.LastIndexAny(os.Args[0], `/\`)+1], filename)
		}
		var byteSlice []byte
		if byteSlice, err = ioutil.ReadFile(filename); err == nil {
			if isForce {
				byteSlice = forceConvertJSONTypeContent(byteSlice)
			}
			content = string(byteSlice)
		}
		isBase64 = false
		return
	}

	err = errors.New("unable to load the config content")
	return
}

func forceConvertJSONTypeContent(srcByteSlice []byte) []byte {
	var dstByteSlice []byte
	if true { //移除第一个"{"前面的所有数据(主要是为了移除BOM头).
		idx := bytes.IndexAny(srcByteSlice, "{")
		if idx < 0 {
			idx = 0
		}
		dstByteSlice = srcByteSlice[idx:]
	}
	if true { //移除最后一个"}"后面的所有数据.
		idx := bytes.LastIndexAny(dstByteSlice, "}")
		if 0 < idx {
			dstByteSlice = dstByteSlice[:idx+1]
		}
	}
	if true { //强制(简单粗暴的)改变Unicode编码的文件内容(Win10可能会默认生成Unicode编码的文件).
		dstByteSlice = bytes.Replace(dstByteSlice, []byte{0x0}, []byte{}, -1)
	}
	return dstByteSlice
}

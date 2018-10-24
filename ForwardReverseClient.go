package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"time"
)

//ForwardReverseClient omit
type ForwardReverseClient struct {
	TransferSlice []*TransferClient
	ForwardSlice  []*ForwardClient
	ReverseSlice  []*ReverseClient
}

func newForwardReverseClientFromContent(s string, isBase64 bool) (cli *ForwardReverseClient, err error) {
	for range "1" {
		var data []byte

		if isBase64 {
			if data, err = base64.StdEncoding.DecodeString(s); err != nil {
				break
			}
		} else {
			data = []byte(s)
		}

		cli = new(ForwardReverseClient)
		if err = json.Unmarshal(data, cli); err != nil {
			break
		}

		//TODO:字段检查
	}

	if err != nil {
		cli = nil
	}

	return
}

func (thls *ForwardReverseClient) run() {
	var totalNum int
	if thls.TransferSlice != nil {
		for _, node := range thls.TransferSlice {
			node.start()
			totalNum++
		}
	}
	if thls.ForwardSlice != nil {
		for _, node := range thls.ForwardSlice {
			node.start()
			totalNum++
		}
	}
	if thls.ReverseSlice != nil {
		for _, node := range thls.ReverseSlice {
			node.start()
			totalNum++
		}
	}
	log.Printf("a total of %v clients and started up.", totalNum)
	for totalNum != 0 {
		time.Sleep(time.Second)
	}
}

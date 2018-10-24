package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/golang/glog"
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

func (thls *ForwardReverseClient) run() (err error) {
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
	if totalNum == 0 {
		err = errors.New("no client can start")
	}
	glog.Warningf("a total of %v clients have been started.", totalNum)
	for err == nil {
		time.Sleep(time.Second)
	}
	return err
}

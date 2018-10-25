package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"time"
)

//TranClientExItem 转发客户端, tran应该有transport/transmit/transfer之意.
//有socket连接到ListenAddr, 我就连接到ServerAddr, 然后让ServerAddr转跳到TargetAddr.
type TranClientExItem struct {
	IsDebug    bool   //处于调试模式
	IsLog      bool   //保存通信日志文件
	ListenAddr string //客户端的监听地址
	ServerAddr string //服务端的地址
	ServerPwd  string //服务端的密码
	TargetAddr string //服务端要转跳到哪个目标地址
	txData     []byte //服务端和客户端之间的通信报文数据
}

//TranClientEx omit
type TranClientEx struct {
	ClientSlice []*TranClientExItem
}

//Start omit
func (thls *TranClientEx) Start() error {
	if len(thls.ClientSlice) == 0 {
		panic(thls.ClientSlice)
	}

	if len(thls.ClientSlice) == 1 {
		return thls.ClientSlice[0].Start()
	}

	for _, item := range thls.ClientSlice {
		go item.Start()
	}
	for true {
		time.Sleep(time.Second)
	}
	return nil
}

func newTranClientExFromContent(s string, isBase64 bool) (cli *TranClientEx, err error) {
	for range "1" {
		var data []byte

		if isBase64 {
			if data, err = base64.StdEncoding.DecodeString(s); err != nil {
				break
			}
		} else {
			data = []byte(s)
		}

		cli = new(TranClientEx)
		if err = json.Unmarshal(data, cli); err != nil {
			break
		}
		if cli.ClientSlice == nil || len(cli.ClientSlice) == 0 {
			err = errors.New("empty configuration")
			break
		}
		for _, item := range cli.ClientSlice {
			msgData := item.ServerPwd + "|" + fmt.Sprintf("%02d", len([]byte(item.TargetAddr))) + "|" + item.TargetAddr
			item.txData = []byte(msgData)
		}
	}

	if err != nil {
		cli = nil
	}

	return
}

//Start omit
func (thls *TranClientExItem) Start() error {
	listener, err := net.Listen("tcp", thls.ListenAddr)
	if err != nil {
		log.Printf("C_Listen, %v, err=%v", thls.ListenAddr, err)
		return err
	}
	log.Printf("Listening on %v", listener.Addr())

	for {
		var sock net.Conn
		if sock, err = listener.Accept(); err != nil {
			if sock != nil {
				sock.Close()
				sock = nil
			}
			log.Printf("C_Accept, %v, err=%v", thls.ListenAddr, err)
			break
		}

		if thls.IsDebug {
			log.Printf("C_Accept, LocalAddr=%v, RemoteAddr=%v, sock=%p", sock.LocalAddr(), sock.RemoteAddr(), sock)
		}

		go thls.doSockProxy(sock)
	}

	listener.Close()

	return err
}

func (thls *TranClientExItem) doSockProxy(sock net.Conn) {
	var serverSock net.Conn
	var err error
	for range "1" {
		if serverSock, err = net.Dial("tcp", thls.ServerAddr); err != nil {
			log.Println(thls.ServerAddr, err)
			break
		}
		var n int
		if n, err = serverSock.Write(thls.txData); (err != nil) || (n != len(thls.txData)) {
			if err == nil {
				err = fmt.Errorf("send part data, %v, %v", len(thls.txData), n)
			}
			break
		}
	}
	if err != nil {
		if thls.IsDebug {
			log.Println(err, thls.ServerAddr)
		}
		if serverSock != nil {
			serverSock.Close()
		}
		sock.Close()
	} else {
		forwardData(sock, serverSock, thls.IsLog)
	}
}

package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net"
)

//TranServer 转发服务器, tran应该有transport/transmit/transfer之意.
//有socket连接到ListenAddr, 我就连接到TargetAddr, 然后把socket的数据转发过去.
type TranServer struct {
	IsDebug    bool   //处于调试模式
	IsLog      bool   //保存通信日志文件
	ListenAddr string //服务器的监听地址
	TargetAddr string //目标地址(有人连接ListenAddr,服务器会将其转到TargetAddr)
}

func newTranServerFromContent(s string, isBase64 bool) (srv *TranServer, err error) {
	for range "1" {
		var data []byte

		if isBase64 {
			if data, err = base64.StdEncoding.DecodeString(s); err != nil {
				break
			}
		} else {
			data = []byte(s)
		}

		srv = new(TranServer)
		if err = json.Unmarshal(data, srv); err != nil {
			break
		}
	}

	if err != nil {
		srv = nil
	}

	return
}

//Start omit
func (thls *TranServer) Start() error {
	listener, err := net.Listen("tcp", thls.ListenAddr)
	if err != nil {
		log.Printf("S_Listen, err=%v", err)
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
			log.Printf("S_Accept, err=%v", err)
			break
		}

		if thls.IsDebug {
			log.Printf("S_Accept, LocalAddr=%v, RemoteAddr=%v, sock=%p", sock.LocalAddr(), sock.RemoteAddr(), sock)
		}

		go thls.doSockProxy(sock)
	}

	listener.Close()

	return err
}

func (thls *TranServer) doSockProxy(sock net.Conn) {
	targetSock, err := net.Dial("tcp", thls.TargetAddr)
	if err != nil {
		log.Println(thls.TargetAddr, err)
		if targetSock != nil {
			targetSock.Close()
		}
		sock.Close()
	} else {
		forwardData(sock, targetSock, thls.IsLog)
	}
}

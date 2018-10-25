package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net"
)

//ListenServer omit
type ListenServer struct {
	IsDebug        bool   //处于调试模式
	IsLog          bool   //保存通信日志文件
	ListenAddrUser string //从哪个地方拿网络数据
	ListenAddrTran string //送网络数据到哪个地方
}

func newListenServerFromContent(s string, isBase64 bool) (srv *ListenServer, err error) {
	for range "1" {
		var data []byte

		if isBase64 {
			if data, err = base64.StdEncoding.DecodeString(s); err != nil {
				break
			}
		} else {
			data = []byte(s)
		}

		srv = new(ListenServer)
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
func (thls *ListenServer) Start() error {
	var err error
	var listenerUser, listenerTran net.Listener

	if listenerUser, err = net.Listen("tcp", thls.ListenAddrUser); err != nil {
		return err
	}

	log.Printf("Listening on %v", listenerUser.Addr())
	if listenerTran, err = net.Listen("tcp", thls.ListenAddrTran); err != nil {
		return err
	}

	log.Printf("Listening on %v", listenerTran.Addr())

	for {
		var uConn, tConn net.Conn
		if uConn, err = listenerUser.Accept(); err != nil {
			if thls.IsDebug {
				log.Println(err)
			}
			continue
		}
		if thls.IsDebug {
			log.Printf("listen, Accept, LocalAddr=%v, RemoteAddr=%v, sock=%p", uConn.LocalAddr(), uConn.RemoteAddr(), uConn)
		}
		if tConn, err = listenerTran.Accept(); err != nil {
			if thls.IsDebug {
				log.Println(err)
			}
			continue
		}
		if thls.IsDebug {
			log.Printf("listen, Accept, LocalAddr=%v, RemoteAddr=%v, sock=%p", tConn.LocalAddr(), tConn.RemoteAddr(), tConn)
		}
		forwardData(uConn, tConn, thls.IsLog)
		uConn.Close()
		tConn.Close()
	}
	//listenerUser.Close()
	//listenerTran.Close()
}

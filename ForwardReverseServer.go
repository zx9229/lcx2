package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net"
)

//ForwardReverseServer omit
type ForwardReverseServer struct {
	Password    string
	ListenAddr  string //net.Listener.Addr()
	listenCache *SafeRsCache
}

func newForwardReverseServerFromContent(s string, isBase64 bool) (srv *ForwardReverseServer, err error) {
	for range "1" {
		var data []byte

		if isBase64 {
			if data, err = base64.StdEncoding.DecodeString(s); err != nil {
				break
			}
		} else {
			data = []byte(s)
		}

		srv = new(ForwardReverseServer)
		if err = json.Unmarshal(data, srv); err != nil {
			break
		}

		//TODO:字段检查
	}

	if err != nil {
		srv = nil
	}

	return
}

func (thls *ForwardReverseServer) initialize() {
	//如果从json反序列化(Unmarshal)出来对象,那么这个对象可能并没有执行完整的初始化,此时需要调用它以执行后续的初始化操作.
	thls.listenCache = newSafeRsCache()
}

func (thls *ForwardReverseServer) run() error {
	thls.initialize()
	//
	var err error
	for range "1" {
		var curListener net.Listener
		if curListener, err = net.Listen("tcp", thls.ListenAddr); err != nil {
			log.Printf("Listen %v with err=%v", thls.ListenAddr, err)
			break
		}
		if curListener != nil {
			log.Printf("Listener(%v) open.", curListener.Addr())
		}
		var sock net.Conn
		for {
			sock = nil
			if sock, err = curListener.Accept(); err != nil {
				break
			}
			//log.Printf("Listener(%v) Accept (%p, L=%v, R=%v)", curListener.Addr(), sock, sock.LocalAddr(), sock.RemoteAddr())
			go thls.handleSocket(sock)
		}
		if curListener != nil {
			log.Printf("Listener(%v) close.", curListener.Addr())
			curListener.Close()
		}
	}
	return err
}

func (thls *ForwardReverseServer) handleSocket(sock net.Conn) {
	buf, isTimeout, err := readDataFromSocket(sock, 0, 1500, true)
	if isTimeout || err != nil {
		log.Printf("readCmdFromSocket failed, sock=%p, isTimeout=%v, err=%v", sock, isTimeout, err)
		sock.Close()
		return
	}
	var obj interface{}
	var objID byte
	if obj, objID, err = buf2msg(buf); err != nil {
		panic(err)
	}
	switch objID {
	case idCmdConnect:
		thls.handleCmdConnect(sock, obj.(*CmdConnect))
	case idCmdListenReq:
		thls.handleCmdListenReq(sock, obj.(*CmdListenReq))
	case idCmdProxyRsp:
		thls.handleCmdProxyRsp(sock, obj.(*CmdProxyRsp))
	default:
		sock.Close()
	}
}

func (thls *ForwardReverseServer) handleCmdConnect(sock net.Conn, dataCmd *CmdConnect) {
	if dataCmd.Pwd != thls.Password {
		sock.Close()
	} else {
		if conn, err := net.Dial("tcp", dataCmd.Addr); err != nil {
			sock.Close()
		} else {
			go forwardData(sock, conn, false)
		}
	}
}

func (thls *ForwardReverseServer) handleCmdListenReq(sock net.Conn, dataReq *CmdListenReq) {
	for range "1" {
		if dataReq.Pwd != thls.Password {
			//不发送"密码错误"消息,如果发送"密码错误"消息,那么它就有特征头了,我就可以写程序暴力破解了.
			sock.Close()
			break
		}
		var err error
		var node *ReverseServer
		if node, err = createReverseServer(sock, dataReq, thls); err != nil { //只要调用这个函数,无论成功失败,都不用管sock了.
			log.Println(err)
			break
		}
		if thls.listenCache.insert(node.listenAddr, node) {
			node.start()
		} else {
			log.Panicln(node)
		}
	}
}

func (thls *ForwardReverseServer) handleCmdProxyRsp(sock net.Conn, dataRsp *CmdProxyRsp) {
	if node, isOk := thls.listenCache.query(dataRsp.Addr); isOk {
		node.feedConn(sock)
	} else {
		sock.Close()
	}
}

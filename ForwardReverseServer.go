package main

import (
	"encoding/base64"
	"encoding/json"
	"net"

	"github.com/golang/glog"
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

func (thls *ForwardReverseServer) run() (err error) {
	thls.initialize()
	//
	for range "1" {
		var curListener net.Listener
		if curListener, err = net.Listen("tcp", thls.ListenAddr); err != nil {
			glog.Errorln(err)
			break
		}
		if curListener != nil {
			glog.Warningf("Listener(%v) open.", curListener.Addr())
		}
		var sock net.Conn
		for {
			sock = nil
			if sock, err = curListener.Accept(); err != nil {
				glog.Errorln(err)
				break
			}
			//glog.Infof("Listener(%v) Accept (%p, R=%v, L=%v)", curListener.Addr(), sock, sock.RemoteAddr(), sock.LocalAddr())
			go thls.handleSocket(sock)
		}
		if curListener != nil {
			glog.Warningf("Listener(%v) close.", curListener.Addr())
			curListener.Close()
		}
	}
	return err
}

func (thls *ForwardReverseServer) handleSocket(sock net.Conn) {
	buf, isTimeout, err := readDataFromSocket(sock, 0, 1500, true)
	if isTimeout || err != nil {
		glog.Warningf("readDataFromSocket failed, isTimeout=%v, err=%v, sock=(%p, R=%v, L=%v)", isTimeout, err, sock, sock.RemoteAddr(), sock.LocalAddr())
		sock.Close()
		return
	}
	var obj interface{}
	var objID byte
	if obj, objID, err = buf2msg(buf); err != nil {
		glog.Errorf("buf2msg failed, objID=%v, err=%v, sock=(%p, R=%v, L=%v)", objID, err, sock, sock.RemoteAddr(), sock.LocalAddr())
		sock.Close()
		return
	}
	switch objID {
	case idCmdConnect:
		thls.handleCmdConnect(sock, obj.(*CmdConnect))
	case idCmdListenReq:
		thls.handleCmdListenReq(sock, obj.(*CmdListenReq))
	case idCmdProxyRsp:
		thls.handleCmdProxyRsp(sock, obj.(*CmdProxyRsp))
	default:
		glog.Errorf("unknown objID=%v, sock=(%p, R=%v, L=%v)", objID, sock, sock.RemoteAddr(), sock.LocalAddr())
		sock.Close()
	}
}

func (thls *ForwardReverseServer) handleCmdConnect(sock net.Conn, dataCmd *CmdConnect) {
	if dataCmd.Pwd != thls.Password {
		glog.Warningf("wrong password in CmdConnect, sock=(%p, R=%v, L=%v)", sock, sock.RemoteAddr(), sock.LocalAddr())
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
			glog.Warningf("wrong password in CmdListenReq, sock=(%p, R=%v, L=%v)", sock, sock.RemoteAddr(), sock.LocalAddr())
			sock.Close()
			break
		}
		var err error
		var node *ReverseServer
		if node, err = createReverseServer(sock, dataReq, thls); err != nil { //只要调用这个函数,无论成功失败,都不用管sock了.
			glog.Warningln(err)
			break
		}
		if thls.listenCache.insert(node.listenAddr, node) {
			node.start()
		} else {
			glog.Fatalln(node)
		}
	}
}

func (thls *ForwardReverseServer) handleCmdProxyRsp(sock net.Conn, dataRsp *CmdProxyRsp) {
	if node, isOk := thls.listenCache.query(dataRsp.Addr); isOk {
		node.feedConn(sock) //它是一个代理用途的socket
	} else {
		sock.Close()
	}
}

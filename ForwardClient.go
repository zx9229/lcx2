package main

import (
	"net"
	"time"

	"github.com/golang/glog"
)

//ForwardClient 正向代理客户端
//我监听ListenAddr,socket连接我的ListenAddr,我就通过ConnectAddr连接服务器,然后让服务器连接TargetAddr
//这样,socket就(通过代理客户端&代理服务器)连接到了TargetAddr
type ForwardClient struct {
	Password    string //json(连接SERVER时需要的密码)
	ListenAddr  string //json(我要监听的地址)
	ConnectAddr string //json(连接SERVER时使用的地址)
	TargetAddr  string //json()
}

func (thls *ForwardClient) start() {
	go thls.listenAndAccept()
}

func (thls *ForwardClient) listenAndAccept() {
	var err error
	var curListener net.Listener
	for curListener == nil {
		if curListener, err = net.Listen("tcp", thls.ListenAddr); err != nil {
			glog.Errorln(err)
			time.Sleep(time.Minute)
		}
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
		glog.Infof("Listener(%v) Accept (%p, R=%v, L=%v)", curListener.Addr(), sock, sock.RemoteAddr(), sock.LocalAddr())
		go thls.handleProxy(sock)
	}
	if curListener != nil {
		glog.Warningf("Listener(%v) close.", curListener.Addr())
	}
	curListener.Close()
}

func (thls *ForwardClient) handleProxy(sock net.Conn) {
	if conn, err := net.Dial("tcp", thls.ConnectAddr); err != nil {
		sock.Close()
	} else {
		if err = writeDataToSocket(conn, msg2buf(&CmdConnect{Addr: thls.TargetAddr})); err != nil {
			conn.Close()
			sock.Close()
		} else {
			forwardData(sock, conn, false)
		}
	}
}

package main

import (
	"net"
	"strings"
	"time"

	"github.com/golang/glog"
)

//EchoClient omit
type EchoClient struct {
	ListenAddr string //监听端口
	WelcomeMsg string //欢迎消息
	EchoHead   string //回显的消息头
	tryOnce    bool
}

func newEchoClientFromContent(listenAddr string, welcomeMsg string, echoHead string) (cli *EchoClient, err error) {
	return &EchoClient{ListenAddr: listenAddr, WelcomeMsg: welcomeMsg, EchoHead: echoHead, tryOnce: true}, nil
}

func (thls *EchoClient) start() {
	go thls.run()
}

func (thls *EchoClient) run() (err error) {
	proxyIt := func(sock net.Conn) {
		if 0 < len(thls.WelcomeMsg) {
			if err = writeDataToSocket(sock, []byte(thls.WelcomeMsg+"\n")); err != nil {
				sock.Close()
				return
			}
		}
		var bufRecv []byte
		var isTimeout bool
		var bufSend []byte
		for {
			if bufRecv, isTimeout, err = readDataFromSocket(sock, '\n', 0, false); isTimeout || (err != nil) {
				break
			}
			bufSend = []byte(strings.Replace(thls.EchoHead, "NOW", time.Now().Format("2006-01-02_15:04:05"), -1))
			bufSend = append(bufSend, bufRecv...)
			if err = writeDataToSocket(sock, bufSend); err != nil {
				break
			}
		}
		sock.Close()
	}
	for range "1" {
		var curListener net.Listener
		for curListener == nil {
			//listenAddr为空时,net库应当有一个隐含操作:随机监听一个可用的端口.程序不准备规避此情况.
			if curListener, err = net.Listen("tcp", thls.ListenAddr); err != nil {
				glog.Errorln(err)
				if thls.tryOnce {
					break
				}
				time.Sleep(time.Minute)
			}
		}
		if curListener == nil {
			break
		}
		glog.Warningf("Listener(%v) open.", curListener.Addr())
		var sock net.Conn
		for {
			sock = nil
			if sock, err = curListener.Accept(); err != nil {
				glog.Errorln(err)
				break
			}
			glog.Infof("Listener(%v) Accept (%p, R=%v, L=%v)", curListener.Addr(), sock, sock.RemoteAddr(), sock.LocalAddr())
			go proxyIt(sock)
		}
		glog.Warningf("Listener(%v) close.", curListener.Addr())
		curListener.Close()
	}
	return
}

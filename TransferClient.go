package main

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/golang/glog"
)

//TransferClient omit
type TransferClient struct {
	ListenAddr string
	TargetAddr string
	tryOnce    bool
}

func newTransferClientFromContent(listenAddr string, targetAddr string) (cli *TransferClient, err error) {
	return &TransferClient{ListenAddr: listenAddr, TargetAddr: targetAddr, tryOnce: true}, nil
}

func (thls *TransferClient) start() {
	go thls.run()
}

func (thls *TransferClient) run() error {
	return transferData(thls.ListenAddr, thls.TargetAddr, thls.tryOnce)
}

func transferData(listenAddr string, targetAddr string, tryOnce bool) (err error) {
	proxyIt := func(sock net.Conn) {
		if conn, eErr := net.Dial("tcp", targetAddr); eErr != nil {
			sock.Close()
		} else {
			forwardData(sock, conn, false)
		}
	}
	for range "1" {
		var curListener net.Listener
		for curListener == nil {
			//listenAddr为空时,net库应当有一个隐含操作:随机监听一个可用的端口.程序不准备规避此情况.
			if curListener, err = net.Listen("tcp", listenAddr); err != nil {
				glog.Errorln(err)
				if tryOnce {
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

func forwardData(conn1 net.Conn, conn2 net.Conn, isLog bool) {
	connCopy := func(conn1 net.Conn, conn2 net.Conn, wg *sync.WaitGroup) {
		//https://github.com/cw1997/NATBypass/blob/master/nb.go
		io.Copy(conn1, conn2)
		conn1.Close()
		wg.Done()
	}
	var wg sync.WaitGroup
	// wait tow goroutines
	wg.Add(2)
	go connCopy(conn1, conn2, &wg)
	go connCopy(conn2, conn1, &wg)
	//blocking when the wg is locked
	wg.Wait()
}

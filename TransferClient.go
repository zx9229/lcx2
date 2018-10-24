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
}

func (thls *TransferClient) start() {
	go transferData(thls.ListenAddr, thls.TargetAddr, true)
}

func transferData(listenAddr string, targetAddr string, doRetry bool) {
	proxyIt := func(sock net.Conn) {
		if conn, err := net.Dial("tcp", targetAddr); err != nil {
			sock.Close()
		} else {
			forwardData(sock, conn, false)
		}
	}
	for range "1" {
		var err error
		var curListener net.Listener
		for curListener == nil {
			if curListener, err = net.Listen("tcp", listenAddr); err != nil {
				glog.Errorln(err)
				if doRetry {
					time.Sleep(time.Minute)
				} else {
					break
				}
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
			glog.Infof("Listener(%v) Accept (%p, L=%v, R=%v)", curListener.Addr(), sock, sock.LocalAddr(), sock.RemoteAddr())
			go proxyIt(sock)
		}
		glog.Warningf("Listener(%v) close.", curListener.Addr())
		curListener.Close()
	}
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

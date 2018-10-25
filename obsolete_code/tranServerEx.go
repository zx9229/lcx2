package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

//TranServerEx 转发服务器, tran应该有transport/transmit/transfer之意.
//有socket连接到ListenAddr, 我根据报文检查密码和TargetAddr, 然后转跳到TargetAddr.
type TranServerEx struct {
	IsDebug     bool   //处于调试模式
	IsLog       bool   //保存通信日志文件
	ListenAddr  string //服务器的监听地址
	Password    string //服务端的密码
	pwdByteData []byte
}

func newTranServerExFromContent(s string, isBase64 bool) (srv *TranServerEx, err error) {
	for range "1" {
		var data []byte

		if isBase64 {
			if data, err = base64.StdEncoding.DecodeString(s); err != nil {
				break
			}
		} else {
			data = []byte(s)
		}

		srv = new(TranServerEx)
		if err = json.Unmarshal(data, srv); err != nil {
			break
		}
		srv.pwdByteData = []byte(srv.Password)
	}

	if err != nil {
		srv = nil
	}

	return
}

//Start omit
func (thls *TranServerEx) Start() error {
	listener, err := net.Listen("tcp", thls.ListenAddr)
	if err != nil {
		log.Printf("S_Listen, %v, err=%v", thls.ListenAddr, err)
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
			log.Printf("S_Accept, %v, err=%v", thls.ListenAddr, err)
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

func (thls *TranServerEx) doSockProxy(sock net.Conn) {
	//密码|targetAddr的长度(2个char)|targetAddr
	var err error
	for range "1" {
		if err = sock.SetReadDeadline(time.Now().Add(time.Millisecond * 3000)); err != nil {
			log.Printf("S_SetReadDeadline, err=%v", err)
			break
		}
		tmpLen := len(thls.pwdByteData)
		tmpBuf := make([]byte, tmpLen)
		var n int
		n, err = sock.Read(tmpBuf)
		isOk := (err == nil) && (n == tmpLen) && simpleEqualCmp(thls.pwdByteData, tmpBuf, n)
		if !isOk {
			if err == nil {
				err = fmt.Errorf("check password fail, n=%v, tmpLen=%v", n, tmpLen)
			}
			break
		}
		tmpLen = 4
		tmpBuf = make([]byte, tmpLen) //【|NN|】
		n, err = sock.Read(tmpBuf)
		isOk = (err == nil) && (n == tmpLen) && (tmpBuf[0] == '|') && (tmpBuf[3] == '|')
		if !isOk {
			err = errors.New("get len fail")
			break
		}
		if tmpLen, err = strconv.Atoi(strings.TrimLeft(string(tmpBuf[1:3]), "0")); (err != nil) || (n <= 0) {
			err = errors.New("convert len fail")
			break
		}
		tmpBuf = make([]byte, tmpLen)
		n, err = sock.Read(tmpBuf)
		sock.SetReadDeadline(time.Time{})
		isOk = (err == nil) && (n == tmpLen)
		if !isOk {
			err = errors.New("get target addr fail")
			break
		}
		targetAddr := string(tmpBuf)
		var targetSock net.Conn
		if targetSock, err = net.Dial("tcp", targetAddr); err != nil {
			log.Println(targetAddr, err)
			if targetSock != nil {
				targetSock.Close()
			}
			break
		}
		forwardData(sock, targetSock, thls.IsLog)
	}
	if err != nil {
		log.Println(err, sock)
		sock.Close()
	}
}

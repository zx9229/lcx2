package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

//ReverseClient 反向Client
//Client和Server建立好通信链接后,如果有socket连接ServerAddr,
//Client会连接Server,然后接手这个socket,
//Client再连接Target,然后将socket的数据转给Target.
type ReverseClient struct {
	Password     string //校验客户端和服务端的密码
	IsDebug      bool   //处于调试模式
	IsLog        bool   //保存通信日志文件
	ServerAddr   string //服务端地址
	TargetAddr   string //目标地址(有人连接ServerAddr,客户端会将其转到TargetAddr)
	Retry        bool   //连接不上服务端,就重试,直到连上为止.
	authByteData []byte //从服务端接收的认证数据
}

func newReverseClientFromContent(s string, isBase64 bool) (cli *ReverseClient, err error) {
	for range "1" {
		var data []byte

		if isBase64 {
			if data, err = base64.StdEncoding.DecodeString(s); err != nil {
				break
			}
		} else {
			data = []byte(s)
		}

		cli = new(ReverseClient)
		if err = json.Unmarshal(data, cli); err != nil {
			break
		}

		//字段检查
		if 0 <= strings.IndexByte(cli.Password, csDelimiter[0]) {
			err = errors.New("Password contains illegal characters")
			break
		}
		cli.authByteData = nil
	}

	if err != nil {
		cli = nil
	}

	return
}

//Start omit
func (thls *ReverseClient) Start() error {
	var err error

	var curConn *CSConn

	isFirstDial := true
	for isFirstDial || thls.Retry {
		if !isFirstDial {
			time.Sleep(time.Second * 5)
		}
		isFirstDial = false

		var sock net.Conn
		if sock, err = net.Dial("tcp", thls.ServerAddr); err != nil {
			if sock != nil {
				sock.Close()
				sock = nil
			}
			log.Printf("C_Dial, err=%v", err)
			continue
		}

		if curConn != nil {
			curConn.Close()
			curConn = nil
		}
		curConn = newCSConn(sock)

		if !thls.doRecvMsgAndCheckPwd(curConn) {
			curConn.Close()
			curConn = nil
			log.Println("C_check, FAIL!")
			continue
		}

		go thls.doHeartbeat(curConn) //启动心跳协程.

		var msg string
		for {
			if msg, err = curConn.readMessage(csDelimiter[0]); err != nil {
				curConn.Close()
				curConn = nil
				break
			}

			if thls.IsDebug {
				log.Printf("S=>C: %v", msg)
			}

			if msg == csSocketReq {
				go thls.doSockProxy(thls.ServerAddr, thls.TargetAddr)
			} else if msg == csHeartbeat {
				//客户端主动心跳,服务端收到心跳后,回复一个心跳,客户端收到心跳回复后,暂不做任何动作.
			}
		}
	}

	return err
}

func (thls *ReverseClient) doSockProxy(addrFrom, addrTo string) {
	var err error

	var connFrom, connTo net.Conn
	for range "1" {
		if connFrom, err = net.Dial("tcp", addrFrom); err != nil {
			log.Println(addrFrom, err)
			break
		}

		if thls.authByteData != nil {
			if num, err := connFrom.Write(thls.authByteData); err != nil || num != len(thls.authByteData) {
				fmt.Println(num, err)
				connFrom.Close()
				connFrom = nil
				break
			}
		}

		if connTo, err = net.Dial("tcp", addrTo); err != nil {
			log.Println(addrTo, err)
			break
		}
	}

	if connFrom != nil && connTo != nil {
		forwardData(connTo, connFrom, thls.IsLog)
	} else {
		if connFrom != nil {
			connFrom.Close()
			connFrom = nil
		}
		if connTo != nil {
			connTo.Close()
			connTo = nil
		}
	}
}

func (thls *ReverseClient) doRecvMsgAndCheckPwd(conn *CSConn) bool {
	var isOk bool

	for range "1" {
		var err error

		if err = conn.writeMessage(thls.Password + csDelimiter); err != nil {
			log.Println(err)
			break
		}

		if err = conn.setReadDeadline(time.Now().Add(time.Millisecond * 2000)); err != nil {
			log.Printf("C_SetReadDeadline, err=%v", err)
			break
		}

		var msg string
		if msg, err = conn.readMessage(csDelimiter[0]); err != nil {
			log.Printf("C_readMessage, err=%v", err)
			break
		}
		if thls.IsDebug {
			log.Printf("S=>C: %v", msg)
		}

		if !strings.HasPrefix(msg, thls.Password) {
			log.Println("ERROR PASSWORD:", msg)
			break
		}

		//客户端解析安全字符串.
		safeData := msg[len(thls.Password) : len(msg)-1]
		thls.authByteData = []byte(safeData)
		if len(thls.authByteData) == 0 {
			thls.authByteData = nil
		}

		if err = conn.rawConn.SetReadDeadline(time.Time{}); err != nil {
			log.Println(err)
			break
		}

		isOk = true
	}

	return isOk
}

func (thls *ReverseClient) doHeartbeat(conn *CSConn) {
	var err error
	for {
		time.Sleep(time.Second * 10)
		if err = conn.writeMessage(csHeartbeat); err != nil {
			conn.Close()
			break
		}
	}
}

package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net"
	"strings"
	"time"
)

//ReverseServer 反向server
//Client和Server建立好通信链接后, 如果有socket连接ListenAddr,
//而且这个socket的RemoteAddr不是ClientHost, 就让客户端接手这个socket,
//Server判断Client的连接的方式: (socket.RemoteAddr == ClientHost)
//有小概率出现"同ClientHost的其它程序连接到ListenAddr"的情况,
//为了辨别这个情况, 引入了AuthData(认证数据),
//如果能从ClientHost的socket中Read到AuthData, 就认为它属于Client,
//此时仍有极小概率误判 ( ClientHost的某程序, 非恶意攻击, 连往ListenAddr, 发送了AuthData )
//同时不再考虑这个小概率事件, 此程序不考虑恶意攻击的情况,
//如果想寻求较高级别的安全, 请搜索 "SSH远程端口转发", 相关命令见下方:
// ssh [-R [bind_address:]port:host:hostport] [-p port] [user@]hostname
type ReverseServer struct {
	Password     string //校验客户端和服务端的密码
	IsDebug      bool   //处于调试模式
	IsLog        bool   //保存通信日志文件
	ListenAddr   string //服务端的监听地址
	ClientHost   string //客户端的IP地址
	AutoAuth     bool   //自动认证(自动生成认证数据[yymmddHHMMSS]共12个字符)
	AuthData     string //认证数据(同ClientHost的其他程序无意中接入ListenAddr)
	authByteData []byte //认证数据转换成的byte切片(认证数据为空时byte切片为nil)
}

func newReverseServerFromContent(s string, isBase64 bool) (srv *ReverseServer, err error) {
	for range "1" {
		var data []byte

		if isBase64 {
			if data, err = base64.StdEncoding.DecodeString(s); err != nil {
				break
			}
		} else {
			data = []byte(s)
		}

		srv = new(ReverseServer)
		if err = json.Unmarshal(data, srv); err != nil {
			break
		}

		//字段检查
		if 0 <= strings.IndexByte(srv.Password, csDelimiter[0]) {
			err = errors.New("Password contains illegal characters")
			break
		}
		if 0 <= strings.IndexByte(srv.AuthData, csDelimiter[0]) {
			err = errors.New("safeData contains illegal characters")
		}
		if srv.AutoAuth {
			srv.AuthData = time.Now().Format("060102150405") //参见该函数的注释文档.
			srv.authByteData = []byte(srv.AuthData)
		} else {
			if 0 < len(srv.AuthData) {
				srv.authByteData = []byte(srv.AuthData)
			} else {
				srv.authByteData = nil
			}
		}
	}

	if err != nil {
		srv = nil
	}

	return
}

//Start omit
func (thls *ReverseServer) Start() error {
	listener, err := net.Listen("tcp", thls.ListenAddr)
	if err != nil {
		log.Printf("S_Listen, err=%v", err)
		return err
	}

	var srvSockChan chan net.Conn //Client连入成功,创建它;Client断开连接,关闭它
	var olClientConn *CSConn

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

		if olClientConn != nil && !olClientConn.working { //其他地方关闭了这个连接.
			olClientConn.Close()
			olClientConn = nil
		}

		isClientHost := guessHost(sock.RemoteAddr()) == thls.ClientHost
		if thls.IsDebug {
			log.Printf("S_client is online(%v)", (olClientConn != nil))
		}

		if isClientHost { //(很可能是)客户端连过来一个socket
			if olClientConn != nil { //服务正常.
				if thls.authByteData != nil { //认证模式
					go thls.doCheckModeSock(sock, srvSockChan)
				} else { //非认证模式
					if 0 < len(srvSockChan) {
						srvSock := <-srvSockChan
						go forwardData(srvSock, sock, thls.IsLog)
					} else { //服务端竟然没有socket需要代理.
						log.Println("abnormal socket")
						sock.Close()
					}
				}
			} else { //服务不可用.
				tmpClientConn := newCSConn(sock)
				if thls.doRecvMsgAndCheckPwd(tmpClientConn) { //校验客户端成功
					olClientConn = tmpClientConn
					srvSockChan = make(chan net.Conn, 20)
					go thls.doRecvMessage(olClientConn)
				} else {
					log.Println("check client sock fail!")
					tmpClientConn.Close()
				}
			}
		} else { //服务器端收到一个socket,它需要被代理
			if olClientConn != nil { //服务正常.
				if err = olClientConn.writeMessage(csSocketReq); err == nil {
					srvSockChan <- sock
				} else { //发送消息失败.
					log.Println(err)
					sock.Close()
					olClientConn.Close()
					close(srvSockChan)
					for i := len(srvSockChan); 0 < i; i-- {
						sock = <-srvSockChan
						sock.Close()
					}
					//无需清理客户端的socket,让它们自行消失就行.
					srvSockChan = nil
					olClientConn = nil
				}
			} else { //服务不可用.
				log.Println("service is not ready yet!")
				sock.Close()
			}
		}
	}

	listener.Close()

	if srvSockChan != nil {
		close(srvSockChan)
		for i := len(srvSockChan); 0 < i; i-- {
			sock := <-srvSockChan
			sock.Close()
		}
	}

	if olClientConn != nil {
		olClientConn.Close()
	}

	return err
}

func simpleEqualCmp(slice1, slice2 []byte, cmpLen int) bool {
	for i := 0; i < cmpLen; i++ {
		if slice1[i] != slice2[i] {
			return false
		}
	}
	return true
}

func (thls *ReverseServer) doCheckModeSock(cliSock net.Conn, srvChan chan net.Conn) {
	checkDataLen := len(thls.authByteData)
	tmpBuf := make([]byte, checkDataLen)
	cliSock.SetReadDeadline(time.Now().Add(time.Millisecond * 1000))
	n, err := cliSock.Read(tmpBuf) //AuthData较短的话,一般一次性就全部发过来了,一般不会被拆成两个包,所以不考虑该情况.
	cliSock.SetReadDeadline(time.Time{})
	if (err == nil) && (n == checkDataLen) && simpleEqualCmp(thls.authByteData, tmpBuf, n) {
		if srvSock, ok := <-srvChan; ok {
			forwardData(srvSock, cliSock, thls.IsLog)
		} else {
			log.Println("chan error", cliSock.LocalAddr(), cliSock.RemoteAddr())
			cliSock.Close()
		}
	} else {
		log.Println("check fail", cliSock.LocalAddr(), cliSock.RemoteAddr())
		cliSock.Close()
		return
	}
}

func (thls *ReverseServer) doRecvMessage(onClientConn *CSConn) {
	var err error
	var msg string
	for {
		if msg, err = onClientConn.readMessage(csDelimiter[0]); err != nil {
			log.Println(err)
			break
		}
		if thls.IsDebug {
			log.Println("C=>S:", msg)
		}
		switch msg {
		case csHeartbeat:
			if err = onClientConn.writeMessage(csHeartbeat); err != nil {
				log.Println(err)
				break
			}
		case csSocketRsp:
			log.Println("C=>S:", csSocketRsp)
		}
	}
	onClientConn.Close()
	log.Println("Client disconnected, service is unavailable!")
}

func (thls *ReverseServer) doRecvMsgAndCheckPwd(conn *CSConn) bool {
	var isOk bool

	for range "1" {
		if thls.ClientHost != guessHost(conn.rawConn.RemoteAddr()) {
			log.Println("invalid client host")
			break
		}

		var err error
		if err = conn.setReadDeadline(time.Now().Add(time.Millisecond * 2000)); err != nil {
			log.Printf("S_SetReadDeadline, err=%v", err)
			break
		}

		var msg string
		if msg, err = conn.readMessage(csDelimiter[0]); err != nil {
			if eErr, ok := err.(net.Error); ok { //Go的类型断言,老是忘记,遂记录在此.
				if eErr.Timeout() {
				}
			}
			log.Printf("S_ReadString, err=%v", err)
			break
		}

		if thls.IsDebug {
			log.Printf("C=>S: %v", msg)
		}

		if msg != thls.Password+csDelimiter {
			log.Println("ERROR PASSWORD:", msg)
			break
		}

		//服务器发送安全字符串给客户端.
		if err = conn.writeMessage(thls.Password + thls.AuthData + csDelimiter); err != nil {
			log.Println(err)
			break
		}

		if err = conn.setReadDeadline(time.Time{}); err != nil {
			log.Println(err)
			break
		}

		isOk = true
	}

	return isOk
}

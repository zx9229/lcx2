package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"
)

func writeDataToSocket(conn net.Conn, buf []byte) error {
	if numWrite, err := conn.Write(buf); err != nil {
		return err
	} else if len(buf) != numWrite {
		return errors.New("bufLen != numWrite")
	} else {
		return nil
	}
}

func readDataFromSocket(conn net.Conn, delim byte, msecTimeout int, doRecover bool) (buf []byte, isTimeout bool, err error) {
	if 0 < msecTimeout { //毫秒级的超时.
		if err = conn.SetReadDeadline(time.Now().Add(time.Duration(msecTimeout) * time.Millisecond)); err != nil {
			return
		}
	}
	buf = make([]byte, 0)
	tmpData := []byte{0}
	numRead := 0
	for {
		if numRead, err = conn.Read(tmpData); err != nil {
			break
		}
		if numRead != 1 { //在逻辑上肯定是等于1的.
			panic(numRead)
		}
		buf = append(buf, tmpData[0])
		if tmpData[0] == delim {
			break
		}
	}
	if (0 < msecTimeout) && doRecover {
		conn.SetReadDeadline(time.Time{})
	}
	if err != nil {
		if netErr, ok := err.(net.Error); ok {
			if netErr.Timeout() {
				isTimeout = true
				err = nil
			}
		}
	}
	return
}

//CmdHeartbeat 心跳命令
type CmdHeartbeat struct {
	DateTime time.Time
}

//CmdConnect CLIENT请求SERVER连接到Addr的命令
type CmdConnect struct {
	Pwd  string
	Addr string
}

//CmdListenReq CLIENT请求SERVER监听Addr的命令
type CmdListenReq struct {
	Pwd  string
	Addr string
}

//CmdListenRsp SERVER监听Addr的结果返回给CLIENT
type CmdListenRsp struct {
	Addr  string
	ErrNo int
}

//CmdProxyReq 在CmdListenReq执行成功,listener接受socket后,向客户端发送该命令,请求客户端创建一个connection以代理那个socket.
type CmdProxyReq struct {
	Addr string
}

//CmdProxyRsp CLIENT代理的结果返回给SERVER
type CmdProxyRsp struct {
	Addr  string
	ErrNo int
}

//准备:Req消息用大写字母,Rsp消息用小写字母,非Req&Rsp对的消息用"非字母的可读字符"(比如(0-9)等.这样调试的时候可能会方便一些)
var (
	idCmdHeartbeat byte = '1'
	idCmdConnect   byte = '2'
	idCmdListenReq byte = 'A'
	idCmdListenRsp byte = 'a'
	idCmdProxyReq  byte = 'B'
	idCmdProxyRsp  byte = 'b'
)

func buf2msg(buf []byte) (obj interface{}, objID byte, err error) {
	//消息格式[objBuf,objID,0](0之前的消息不允许为0,所以json的内容不可以含0)
	bufLen := len(buf)
	if bufLen <= 2 || buf[bufLen-1] != 0 {
		err = errors.New("illegal message")
		return
	}
	objID = buf[bufLen-2]
	switch objID {
	case idCmdHeartbeat:
		obj = new(CmdHeartbeat)
	case idCmdConnect:
		obj = new(CmdConnect)
	case idCmdListenReq:
		obj = new(CmdListenReq)
	case idCmdListenRsp:
		obj = new(CmdListenRsp)
	case idCmdProxyReq:
		obj = new(CmdProxyReq)
	case idCmdProxyRsp:
		obj = new(CmdProxyRsp)
	default:
		err = fmt.Errorf("unknown objectID=%v", objID)
	}
	err = json.Unmarshal(buf[:bufLen-2], obj)
	return
}

func msg2buf(v interface{}) []byte {
	//请传入结构体的指针.
	var err error
	var objID byte
	var objBuf []byte
	switch v.(type) {
	case *CmdHeartbeat:
		objID = idCmdHeartbeat
		objBuf, err = json.Marshal(v)
	case *CmdConnect:
		objID = idCmdConnect
		objBuf, err = json.Marshal(v)
	case *CmdListenReq:
		objID = idCmdListenReq
		objBuf, err = json.Marshal(v)
	case *CmdListenRsp:
		objID = idCmdListenRsp
		objBuf, err = json.Marshal(v)
	case *CmdProxyReq:
		objID = idCmdProxyReq
		objBuf, err = json.Marshal(v)
	case *CmdProxyRsp:
		objID = idCmdProxyRsp
		objBuf, err = json.Marshal(v)
	default:
		panic(v)
	}
	if err != nil {
		panic(err)
	}
	//消息格式[objBuf,objID,0](0之前的消息不允许为0,所以json的内容不可以含0)
	return append(objBuf, []byte{objID, 0}...)
}

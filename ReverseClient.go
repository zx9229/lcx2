package main

import (
	"net"
	"time"

	"github.com/golang/glog"
)

//ReverseClient 反向客户端
type ReverseClient struct {
	Password    string    //json([反向SERVER端]的密码)
	ConnectAddr string    //json(连接到[反向SERVER端])
	SrvLisAddr  string    //json(让[反向SERVER端]监听的[proxyPort])
	TargetAddr  string    //json(代理[从proxyPort接受的socket]到目标地址)
	cliConn     *SafeConn //连接到SERVER的connection
	tmHeartbeat time.Time //心跳时刻
}

func (thls *ReverseClient) start() {
	go thls.reconnectWork()
}

func (thls *ReverseClient) reconnectWork() {
	//只允许本函数修改(thls.cliConn)的值.
	if thls.cliConn != nil {
		thls.cliConn.Close()
		thls.cliConn = nil
	}
	var err error
	var conn net.Conn
	for {
		if conn, err = net.Dial("tcp", thls.ConnectAddr); err != nil {
			time.Sleep(time.Second * 5)
			continue
		}
		thls.cliConn = newSafeConn(conn)
		thls.tmHeartbeat = time.Now()
		if err = thls.cliConn.WriteBytes(msg2buf(&CmdListenReq{Pwd: thls.Password, Addr: thls.SrvLisAddr})); err != nil {
			thls.cliConn.Close()
			thls.cliConn = nil
			time.Sleep(time.Second * 30)
			continue
		}
		break
	}
	go thls.heartbeatWork()
	go thls.eventWork()
}

func (thls *ReverseClient) heartbeatWork() {
	//断线并重连之后,会启动新协程,此时就需要老协程自动退出.
	curCliConn := thls.cliConn
	if curCliConn == nil {
		return
	}
	var err error
	var tmCheck time.Time
	for curCliConn == thls.cliConn {
		time.Sleep(time.Minute)
		tmCheck = time.Now()
		if err = curCliConn.WriteBytes(msg2buf(&CmdHeartbeat{DateTime: tmCheck})); err != nil {
			//glog.Infoln(err) //一般让recv的那个线程打印错误,这样错误较少重复.
			break
		} else if 180 < tmCheck.Sub(thls.tmHeartbeat).Seconds() {
			glog.Warningf("heartbeat timeout, tmCheck=%v, tmHeartbeat=%v, sock=(%p, R=%v, L=%v)", tmCheck, thls.tmHeartbeat, curCliConn.rawConn, curCliConn.RemoteAddr(), curCliConn.LocalAddr())
			break
		}
	}
	curCliConn.Close()
}

func (thls *ReverseClient) eventWork() {
	var err error
	var buf []byte
	var obj interface{}
	var objID byte
	for {
		if buf, err = thls.cliConn.ReadBytes(); err != nil {
			glog.Warningln(err)
			break
		}
		if obj, objID, err = buf2msg(buf); err != nil {
			glog.Errorf("buf2msg failed, objID=%v, err=%v, sock=(%p, R=%v, L=%v)", objID, err, thls.cliConn.rawConn, thls.cliConn.RemoteAddr(), thls.cliConn.LocalAddr())
			thls.cliConn.Close()
			break
		}
		switch objID {
		case idCmdHeartbeat:
			thls.tmHeartbeat = time.Now() //收到了SERVER的心跳.
		case idCmdListenRsp:
			thls.handleCmdListenRsp(obj.(*CmdListenRsp))
		case idCmdProxyReq:
			//glog.Infoln("CmdProxyReq", obj)
			go thls.handleCmdProxyReq(obj.(*CmdProxyReq))
		default:
		}
	}
	thls.cliConn.Close()
	time.Sleep(time.Second * 5) //如果client和server的密码不一致,那么server会kill掉client,此时加上5秒的间隔,不至于暴力重连.
	go thls.reconnectWork()
}

func (thls *ReverseClient) handleCmdListenRsp(dataRsp *CmdListenRsp) {
	if dataRsp.ErrNo == 0 {
		glog.Infoln("CmdListenRsp", dataRsp)
	} else {
		glog.Warningln("CmdListenRsp", dataRsp)
		thls.cliConn.Close()
		//SERVER那边listen一个(Host:Port)失败了,要么直接放弃,要么过会再试,
		//我选择间隔一段时间之后,再试一试.
		time.Sleep(time.Minute)
	}
}

func (thls *ReverseClient) handleCmdProxyReq(dataReq *CmdProxyReq) {
	var err error
	var pConn, tConn net.Conn
	dataRsp := &CmdProxyRsp{Addr: dataReq.Addr, ErrNo: 0}
	if pConn, err = net.Dial("tcp", thls.ConnectAddr); err != nil {
		dataRsp.ErrNo = -1
		if err = thls.cliConn.WriteBytes(msg2buf(dataRsp)); err != nil {
			thls.cliConn.Close()
		}
	} else {
		if err = writeDataToSocket(pConn, msg2buf(dataRsp)); err != nil { //我没写错,就是往这个proxyConn里面写一条消息.
			pConn.Close()
		} else {
			if tConn, err = net.Dial("tcp", thls.TargetAddr); err != nil {
				pConn.Close()
			} else {
				forwardData(pConn, tConn, false)
			}
		}
	}
}

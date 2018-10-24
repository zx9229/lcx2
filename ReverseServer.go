package main

import (
	"net"
	"sync"
	"time"

	"github.com/golang/glog"
)

//ReverseServer omit
type ReverseServer struct {
	Password    string //json([反向SERVER端]的密码)
	listenAddr  string //net.Listener.Addr()(HOST:PORT)
	listener    net.Listener
	cliConn     *SafeConn    //客户端的连接,发送(L:HOST:PORT)
	connQue     *SafeConnQue //accepted的(需要代理的)连接.
	tmHeartbeat time.Time    //心跳时刻
	once        sync.Once
	frs         *ForwardReverseServer //它里面缓存了数据,关闭时,需要清理对应的数据.
}

//不论成功还是失败,都不用管clientConn了.
func createReverseServer(clientConn net.Conn, dataReq *CmdListenReq, frs *ForwardReverseServer) (node *ReverseServer, err error) {
	for range "1" {
		dataRsp := &CmdListenRsp{Addr: dataReq.Addr, ErrNo: 0}
		var tmpListener net.Listener
		if tmpListener, err = net.Listen("tcp", dataReq.Addr); err != nil {
			dataRsp.ErrNo = -1
			if tmpErr := writeDataToSocket(clientConn, msg2buf(dataRsp)); tmpErr != nil {
				clientConn.Close()
			} else {
				//监听失败,理论上,客户端收到失败消息后,会主动关闭自己,这里等待一段时间,如果客户端不主动关闭,就由服务端关掉它.
				go func() {
					time.Sleep(time.Second * 10)
					clientConn.Close()
				}()
			}
			break
		}
		if tmpListener != nil {
			glog.Warningf("Listener(%v) open.", tmpListener.Addr())
		}
		if err = writeDataToSocket(clientConn, msg2buf(dataRsp)); err != nil {
			if tmpListener != nil {
				glog.Warningf("Listener(%v) close.", tmpListener.Addr())
				tmpListener.Close()
				tmpListener = nil
			}
			clientConn.Close()
			break
		}
		node = &ReverseServer{listenAddr: dataReq.Addr, listener: tmpListener, cliConn: newSafeConn(clientConn), connQue: newSafeConnQue(), frs: frs}
	}
	return
}

func (thls *ReverseServer) start() {
	go thls.heartbeatWork()
	go thls.eventWork()
	go thls.acceptWork()
}

func (thls *ReverseServer) acceptWork() {
	var err error
	var sock net.Conn
	dataReq := msg2buf(&CmdProxyReq{Addr: thls.listenAddr})
	for {
		sock = nil
		if sock, err = thls.listener.Accept(); err != nil {
			glog.Errorln(err)
			break
		}
		//glog.Infof("Listener(%v) Accept (%p, R=%v, L=%v)", curListener.Addr(), sock, sock.RemoteAddr(), sock.LocalAddr())
		thls.connQue.pushBack(sock)
		if err = thls.cliConn.WriteBytes(dataReq); err != nil {
			break
		}
	}
	thls.stop()
}

func (thls *ReverseServer) heartbeatWork() {
	var err error
	var tmCheck time.Time
	for {
		time.Sleep(time.Minute)
		tmCheck = time.Now()
		if err = thls.cliConn.WriteBytes(msg2buf(&CmdHeartbeat{DateTime: tmCheck})); err != nil {
			//glog.Infoln(err) //一般让recv的那个线程打印错误,这样错误较少重复.
			break
		} else if 180 < tmCheck.Sub(thls.tmHeartbeat).Seconds() {
			glog.Warningf("heartbeat timeout, tmCheck=%v, tmHeartbeat=%v, sock=(%p, R=%v, L=%v)", tmCheck, thls.tmHeartbeat, thls.cliConn.rawConn, thls.cliConn.RemoteAddr(), thls.cliConn.LocalAddr())
			break
		}
	}
	thls.stop()
}

func (thls *ReverseServer) eventWork() {
	//目的:尽可能早的拿到"连接断开了"事件.
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
			thls.tmHeartbeat = time.Now() //收到了CLIENT的心跳.
		case idCmdProxyRsp:
			thls.handleCmdProxyRsp(obj.(*CmdProxyRsp))
		default:
		}
	}
	thls.stop()
}

func (thls *ReverseServer) handleCmdProxyRsp(dataRsp *CmdProxyRsp) {
	if dataRsp.ErrNo != 0 {
		glog.Warningln("CmdProxyRsp", dataRsp)
		//客户端代理失败了一次,服务端就会有一个连接无法代理,所以服务端要销毁一个连接.
		if tmpConn, isOk := thls.connQue.popFront(); isOk {
			tmpConn.Close()
		}
	}
}

func (thls *ReverseServer) stop() {
	thls.once.Do(func() {
		if thls.listener != nil {
			glog.Warningf("Listener(%v) close.", thls.listener.Addr())
			thls.listener.Close()
		}
		thls.cliConn.Close()
		if val, isOk := thls.frs.listenCache.delete(thls.listenAddr); !isOk || thls != val {
			glog.Fatalln(isOk, val, thls)
		}
	})
	thls.connQue.clear(true)
}

func (thls *ReverseServer) feedConn(conn net.Conn) {
	if aConn, isOk := thls.connQue.popFront(); isOk {
		go forwardData(conn, aConn, false)
	} else {
		conn.Close()
	}
	//逻辑上来说,本端accept一个socket,然后发给对端一个消息,对端connect过来一个socket,然后通过feedConn送过来,
	//所以,应当能取到数据才对,唯一取不到数据的情况是:刚发给对端一个消息,本端和对端断开连接,然后socket通过feedConn送过来了.
}

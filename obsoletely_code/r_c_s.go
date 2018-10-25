package main

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

//LCX2Interface omit
type LCX2Interface interface {
	Start() error
}

const (
	csDelimiter = "|"
	csHeartbeat = "HeartBeat|" //心跳
	csSocketReq = "SocketReq|" //服务端请求客户端创建连接
	csSocketRsp = "SocketRsp|" //客户端连往服务端执行结束
)

/*
[客户端<=>服务器]通信过程
client连接server
client发送: 密码+|
server检查密码通过
server发送: 密码+安全码+|
client接收: 密码+安全码+|
client校验密码,缓存安全码
*/

//CSConn Client和Server的连接.
type CSConn struct {
	rawConn net.Conn
	r       *bufio.Reader //只允许调用它读数据.
	mtx     *sync.Mutex
	working bool
}

func newCSConn(conn net.Conn) *CSConn {
	curData := &CSConn{rawConn: conn, r: bufio.NewReader(conn), mtx: new(sync.Mutex), working: true}
	return curData
}

func (thls *CSConn) setReadDeadline(t time.Time) error {
	return thls.rawConn.SetReadDeadline(t)
}

func (thls *CSConn) readMessage(delim byte) (string, error) {
	return thls.r.ReadString(delim)
}

func (thls *CSConn) writeMessage(mesg string) error { //只允许调用它发送数据,不允许调用(net.Conn)
	byteMesg := []byte(mesg)
	thls.mtx.Lock()
	defer thls.mtx.Unlock()
	n, err := thls.rawConn.Write(byteMesg)
	if err != nil {
		return err
	}
	if len(byteMesg) != n {
		return errors.New("send data fail")
	}
	return err
}

//Close omit
func (thls *CSConn) Close() {
	//TODO:不知道 bufioReader 还要不要关闭.
	thls.rawConn.Close()
	thls.working = false
}

////////////////////////////////////////////////////////////

func guessHost(addr net.Addr) string {
	var idx int
	if idx = strings.LastIndex(addr.String(), ":"); idx <= 0 {
		return ""
	}
	return addr.String()[:idx]
}

func connCopy(conn1 net.Conn, conn2 net.Conn, wg *sync.WaitGroup) {
	//https://github.com/cw1997/NATBypass/blob/master/nb.go
	io.Copy(conn1, conn2)
	conn1.Close()
	wg.Done()
}

func forward(conn1 net.Conn, conn2 net.Conn) {
	var wg sync.WaitGroup
	// wait tow goroutines
	wg.Add(2)
	go connCopy(conn1, conn2, &wg)
	go connCopy(conn2, conn1, &wg)
	//blocking when the wg is locked
	wg.Wait()
}

//forwardData 如果用它产生log文件,那么会用conn1的net.Addr进行命名.
func forwardData(conn1 net.Conn, conn2 net.Conn, isLog bool) {

	funIoCopy := func(conn1 net.Conn, conn2 net.Conn, wg *sync.WaitGroup, w io.Writer) {
		if w != nil {
			mw := io.MultiWriter(conn1, w)
			io.Copy(mw, conn2)
		} else {
			io.Copy(conn1, conn2)
		}
		conn1.Close()
		wg.Done()
	}

	funGetChildLog := func(nlfObj *NetLogFile, isA2B bool) io.Writer {
		if nlfObj == nil {
			return nil
		}
		if isA2B {
			return nlfObj.GetChildFileA2B()
		}
		return nlfObj.GetChildFileB2A()
	}

	var wg sync.WaitGroup
	// wait two goroutines
	wg.Add(2)

	var nlf *NetLogFile

	if isLog {
		filename := RecommendFilename(conn1.LocalAddr(), conn1.RemoteAddr())
		var err error
		if nlf, err = newNetLogFile(filename); err != nil {
			nlf = nil
			log.Println("newNetLogFile", filename, err)
		}
	}

	go funIoCopy(conn1, conn2, &wg, funGetChildLog(nlf, true))
	go funIoCopy(conn2, conn1, &wg, funGetChildLog(nlf, false))

	//blocking when the wg is locked
	wg.Wait()

	if nlf != nil {
		nlf.Close()
	}
}

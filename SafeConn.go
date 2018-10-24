package main

import (
	"net"
	"sync"
)

//SafeConn 多线程同时read/write时是安全的.
type SafeConn struct {
	mutexRead  sync.Mutex
	mutexWrite sync.Mutex
	rawConn    net.Conn
}

//newSafeConn omit
func newSafeConn(conn net.Conn) *SafeConn {
	return &SafeConn{rawConn: conn}
}

// WriteBytes reads data from the connection.
func (thls *SafeConn) WriteBytes(buf []byte) (err error) {
	thls.mutexWrite.Lock()
	err = writeDataToSocket(thls.rawConn, buf)
	thls.mutexWrite.Unlock()
	return
}

// ReadBytes writes data to the connection.
func (thls *SafeConn) ReadBytes() (buf []byte, err error) {
	var isTimeout bool
	thls.mutexRead.Lock()
	buf, isTimeout, err = readDataFromSocket(thls.rawConn, 0, 0, false)
	thls.mutexRead.Unlock()
	if isTimeout {
		panic(isTimeout) //超时设为0就应当不超时才对.
	}
	return
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (thls *SafeConn) Close() {
	thls.mutexWrite.Lock()
	thls.rawConn.Close()
	thls.mutexWrite.Unlock()
}

// LocalAddr returns the local network address.
func (thls *SafeConn) LocalAddr() net.Addr {
	return thls.rawConn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (thls *SafeConn) RemoteAddr() net.Addr {
	return thls.rawConn.RemoteAddr()
}

package main

import (
	"net"
	"sync"
)

//SafeConnQue omit
type SafeConnQue struct {
	sync.Mutex
	S []net.Conn
}

func newSafeConnQue() *SafeConnQue {
	return &SafeConnQue{S: make([]net.Conn, 0)}
}

func (thls *SafeConnQue) pushBack(node net.Conn) {
	thls.Lock()
	thls.S = append(thls.S, node)
	thls.Unlock()
}

func (thls *SafeConnQue) popFront() (node net.Conn, isExists bool) {
	thls.Lock()
	if 0 < len(thls.S) {
		node = thls.S[0]
		thls.S = thls.S[1:]
		isExists = true
	}
	thls.Unlock()
	return
}

func (thls *SafeConnQue) clear(doClose bool) {
	thls.Lock()
	if doClose {
		for _, node := range thls.S {
			if node != nil {
				node.Close()
			}
		}
	}
	thls.S = make([]net.Conn, 0)
	thls.Unlock()
}

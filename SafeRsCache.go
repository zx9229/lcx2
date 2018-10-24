package main

import (
	"sync"
)

//SafeRsCache 安全的ReverseServer缓存类.
type SafeRsCache struct {
	sync.Mutex
	M map[string]*ReverseServer
}

func newSafeRsCache() *SafeRsCache {
	return &SafeRsCache{M: make(map[string]*ReverseServer)}
}

func (thls *SafeRsCache) insert(key string, val *ReverseServer) bool {
	var isSuccess bool
	thls.Lock()
	if _, isSuccess = thls.M[key]; !isSuccess {
		thls.M[key] = val
	}
	isSuccess = !isSuccess
	thls.Unlock()
	return isSuccess
}

func (thls *SafeRsCache) query(key string) (val *ReverseServer, isExists bool) {
	thls.Lock()
	val, isExists = thls.M[key]
	thls.Unlock()
	return
}

func (thls *SafeRsCache) delete(key string) (val *ReverseServer, isSuccess bool) {
	thls.Lock()
	if val, isSuccess = thls.M[key]; isSuccess {
		delete(thls.M, key)
	}
	thls.Unlock()
	return
}

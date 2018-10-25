package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
)

//SlaveClient omit
//例如,我们需要互联网访问一台内网机器的SSH服务,我们有一台公网IP是8.8.8.8的机器,
//如果内网机器允许程序监听端口,那么建议使用r_server+r_client的组合
//如果内网机器禁止程序监听端口,那么可以使用slave,
//建议ServerAddr配置8.8.8:port,TargetAddr配置127.0.0.1:port,这样更好.详见下方注释.
type SlaveClient struct {
	IsDebug    bool   //处于调试模式
	IsLog      bool   //保存通信日志文件
	ServerAddr string //从哪个地方拿网络数据
	TargetAddr string //送网络数据到哪个地方
}

func newSlaveClientFromContent(s string, isBase64 bool) (cli *SlaveClient, err error) {
	for range "1" {
		var data []byte

		if isBase64 {
			if data, err = base64.StdEncoding.DecodeString(s); err != nil {
				break
			}
		} else {
			data = []byte(s)
		}

		cli = new(SlaveClient)
		if err = json.Unmarshal(data, cli); err != nil {
			break
		}
	}

	if err != nil {
		cli = nil
	}

	return
}

//Start omit
//它先连接ServerAddr, 成功之后, 再连接TargetAddr, 然后转发两个conn的数据,
//我感觉, ServerAddr配置自己的服务, TargetAddr配置别人家的服务, 这样稍微好一些,
//如果自己的服务都连不上, 就不要搞别人家的服务了嘛.
func (thls *SlaveClient) Start() error {
	for {
		var err error
		var sConn net.Conn
		var tConn net.Conn
		for {
			if sConn, err = net.Dial("tcp", thls.ServerAddr); err != nil {
				if thls.IsDebug {
					log.Println(err)
				}
				time.Sleep(time.Second * 5)
				continue
			}
			if thls.IsDebug {
				log.Println(fmt.Sprintf("Dial ServerAddr success, LocalAddr=%v, RemoteAddr=%v", sConn.LocalAddr(), sConn.RemoteAddr()))
			}
			break
		}
		for {
			if tConn, err = net.Dial("tcp", thls.TargetAddr); err != nil {
				if thls.IsDebug {
					log.Println(err)
				}
				time.Sleep(time.Second * 5)
				continue
			}
			if thls.IsDebug {
				log.Println(fmt.Sprintf("Dial TargetAddr success, LocalAddr=%v, RemoteAddr=%v", tConn.LocalAddr(), tConn.RemoteAddr()))
			}
			break
		}
		forwardData(sConn, tConn, thls.IsLog)
	}
}

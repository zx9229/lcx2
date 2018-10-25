# lcx2

* C实现的lcx  
[powerhacker/lcx](https://github.com/powerhacker/lcx)。  

* golang实现的lcx  
[cw1997/NATBypass: 一款lcx在golang下的实现](https://github.com/cw1997/NATBypass)。  

* 我拷贝并修改了`cw1997/NATBypass`，同时增加了"反向代理"的功能。

## Golang在windows下交叉编译linux程序
* 下载源码
```
go get -u -v github.com/zx9229/lcx2
```
* Golang在windows下交叉编译linux程序
```cmd
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build
```

## 使用说明  

* 查看某个`type`的配置  
`./lcx2.exe -help -type=TYPE`，比如`.\lcx2.exe -help -type=r_client`。

* 各`type`的说明  
`r_client`参见[r_client.go](https://github.com/zx9229/lcx2/blob/master/r_client.go)顶部的注释。  
`r_server`参见[r_server.go](https://github.com/zx9229/lcx2/blob/master/r_server.go)顶部的注释。  

* 使用示例  
`.\lcx2.exe -type=TYPE -file=.\cfg.json`

## TODO:
准备添加“复用监听端口的socket反向socket转发器”功能。例如：  
server模式下，监听端口A，用于client连接。  
client以操作员模式登录成功，执行`listen host:portB`动态的让server监听端口B。  
client连接之后，校验成功，发送[host:portB]转跳到B的心跳模式。  
有程序连接server的端口B，server向客户端发消息`建立一个到B的链接`，  
客户端连接端口A，然后发消息`我要跳到B`，然后跳到B，完成端口对接。  
这样的话，只要有一个端口可以监听，我们就能让被限机器访问很多服务。  
同样，可以写一个“复用监听端口的正向socket转发器”。  
通过client模式+server模式，我们可以变相实现“只要机器开放了一个公网端口，它就开放了所有公网端口”。  

# lcx2

* C实现的lcx  
[powerhacker/lcx](https://github.com/powerhacker/lcx)。  

* golang实现的lcx  
[cw1997/NATBypass: 一款lcx在golang下的实现](https://github.com/cw1997/NATBypass)。  

* 我拷贝了`cw1997/NATBypass`的部分代码，同时增加了"反向代理"的功能。


## Golang在windows下交叉编译linux程序

* 下载源码
```
go get -u -v github.com/zx9229/lcx2
```

* Golang在windows下交叉编译linux程序
```bat
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build
```


## TODO

日志功能(我目前没有需求,所以没有动力增加它)。

package main

import (
	"flag"
	"fmt"

	"github.com/golang/glog"
)

//lcx2Interface omit
type lcx2Interface interface {
	run() error
}

func main() {
	if true {
		//文件(github.com/golang/glog/glog.go)有各参数的说明.
		//stderrthreshold: [INFO|WARNING|ERROR|FATEL],
		//日志写入文件中,如果日志的等级达到阈值,就顺便把日志输出到标准错误.
		//如果启用了"logtostderr"参数,则"stderrthreshold"(好像)无效.
		flag.Set("stderrthreshold", "WARNING")
		//https://blog.csdn.net/lengyuezuixue/article/details/79203165
		flag.Set("logtostderr", "true")
	}
	var (
		argHelp   bool
		argType   string
		argListen string
		argTarget string
		argConf   string
		argOffset bool
		argForce  bool
		argBase64 string
		argStdin  bool
	)
	flag.BoolVar(&argHelp, "help", false, "[M] show this help.")
	flag.StringVar(&argType, "type", "tran", "[M] tran, client, server")
	flag.StringVar(&argListen, "listen", "", "[M][tran] listenAddr")
	flag.StringVar(&argTarget, "target", "", "[M][tran] targetAddr")
	flag.StringVar(&argConf, "conf", "", "[M] configuration file name.")
	flag.BoolVar(&argOffset, "offset", false, "[M] find the conf based on the dir where the exe is located.")
	flag.BoolVar(&argForce, "force", false, "[M] force parsing json-type conf files in a simple and rude manner.")
	flag.StringVar(&argBase64, "base64", "", "[M] base64 encoded data for the configuration file.")
	flag.BoolVar(&argStdin, "stdin", false, "[M] read base64 encoded data from standard input.")
	flag.Parse()
	//
	for range "1" {
		if argHelp {
			flag.Usage()
			showHelp(argType)
			break
		}
		var err error
		var content string
		var isBase64 bool
		if argType != "tran" {
			content, isBase64, err = loadConfigContent(argStdin, argBase64, argConf, argForce, argOffset)
			if err != nil {
				glog.Errorf("loadConfigContent with err=%v", err)
				break
			}
		}
		var lcx2Obj lcx2Interface
		switch argType {
		case "client":
			lcx2Obj, err = newForwardReverseClientFromContent(content, isBase64)
		case "server":
			lcx2Obj, err = newForwardReverseServerFromContent(content, isBase64)
		case "tran":
			lcx2Obj, err = newTransferClientFromContent(argListen, argTarget)
		default:
			err = fmt.Errorf("unknown type=%v", argType)
		}
		if err != nil {
			glog.Errorf("new object with err=%v", err)
			break
		}
		if err = lcx2Obj.run(); err != nil {
			glog.Errorf("run object with err=%v", err)
			break
		}
	}
	glog.Flush()
}

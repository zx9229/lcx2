package main

import (
	"flag"

	"github.com/golang/glog"
)

func main() {
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
	if argHelp {
		flag.Usage()
		return
	}
	//
	if argType == "tran" {
		transferData(argListen, argTarget, false)
		return
	}
	//
	content, isBase64, err := loadConfigContent(argStdin, argBase64, argConf, argForce, argOffset)
	if err != nil {
		glog.Fatalln(err)
	}
	//
	if argType == "client" {
		if obj, err := newForwardReverseClientFromContent(content, isBase64); err != nil {
			glog.Fatalln(err)
		} else {
			obj.run()
		}
	} else if argType == "server" {
		if obj, err := newForwardReverseServerFromContent(content, isBase64); err != nil {
			glog.Fatalln(err)
		} else {
			obj.run()
		}
	} else {
		glog.Fatalln("unknown type", argType)
	}
}

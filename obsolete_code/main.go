package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
)

func main() {

	var (
		isHelp     bool
		isStdin    bool
		base64Data string
		confName   string
		isForce    bool
		isOffset   bool
		typeData   string
	)

	flag.BoolVar(&isHelp, "help", false, "show this help.")
	flag.BoolVar(&isStdin, "stdin", false, "read base64 encoded data from standard input.")
	flag.StringVar(&base64Data, "base64", "", "base64 encoded data for the configuration file.")
	flag.StringVar(&confName, "conf", "", "configuration file name.")
	flag.BoolVar(&isForce, "force", false, "force parsing json-type conf files in a simple and rude manner.")
	flag.BoolVar(&isOffset, "offset", false, "find the conf based on the dir where the exe is located.")
	//如果配置文件里面有配置路径, 还是一个相对路径, 那么这个路径还是根据工作目录走的, 不是根据程序所在的目录走的.
	flag.StringVar(&typeData, "type", "", "r_client, r_server, tran, t_client, t_server, slave, listen")
	flag.Parse()

	if isHelp {
		flag.Usage()
		fmt.Println()
		fmt.Println(calcConfigInfo(typeData))
		fmt.Println()
		return
	}

	content, isBase64, err := loadConfigContent(isStdin, base64Data, confName, isForce, isOffset)
	if err != nil {
		log.Fatalln("loadConfigContent,", err)
	}

	var lcx2Obj LCX2Interface

	switch typeData {
	case "r_client":
		lcx2Obj, err = newReverseClientFromContent(content, isBase64)
	case "r_server":
		lcx2Obj, err = newReverseServerFromContent(content, isBase64)
	case "tran":
		lcx2Obj, err = newTranServerFromContent(content, isBase64)
	case "t_client":
		lcx2Obj, err = newTranClientExFromContent(content, isBase64)
	case "t_server":
		lcx2Obj, err = newTranServerExFromContent(content, isBase64)
	case "slave":
		lcx2Obj, err = newSlaveClientFromContent(content, isBase64)
	case "listen":
		lcx2Obj, err = newListenServerFromContent(content, isBase64)
	default:
		log.Fatalf("unknown type=%v", typeData)
	}

	if err != nil {
		log.Fatalln(err)
	}

	if err = lcx2Obj.Start(); err != nil {
		log.Fatalln(err)
	}
}

func calcConfigInfo(typeData string) string {
	var bytes []byte
	switch typeData {
	case "r_client":
		bytes, _ = json.Marshal(ReverseClient{})
	case "r_server":
		bytes, _ = json.Marshal(ReverseServer{})
	case "tran":
		bytes, _ = json.Marshal(TranServer{})
	case "t_client":
		bytes, _ = json.Marshal(TranClientEx{ClientSlice: []*TranClientExItem{new(TranClientExItem)}})
	case "t_server":
		bytes, _ = json.Marshal(TranServerEx{})
	case "slave":
		bytes, _ = json.Marshal(SlaveClient{})
	case "listen":
		bytes, _ = json.Marshal(ListenServer{})
	default:
		bytes = []byte("unknown type=" + typeData)
	}
	return string(bytes)
}

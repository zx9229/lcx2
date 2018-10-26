package main

import (
	"fmt"
)

func showHelp(argType string) {
	fmt.Println()
	fmt.Println("Version20181025")
	fmt.Println("Example:")
	fmt.Println("./exe -type tran   -listen 127.0.0.1:31433 -target 127.0.0.1:1433")
	fmt.Println("./exe -type server -conf cfg_server.json")
	fmt.Println("./exe -type client -conf cfg_client.json")
	switch argType {
	case "server":
		fmt.Println(configServer())
	case "client":
		fmt.Println(configClient())
	default:
	}
	fmt.Println()
}

func configServer() string {
	content := `
{
    "Password": "PWD",
    "ListenAddr": "localhost:10254"
}`
	return content
}

func configClient() string {
	content := `
{
    "TransferSlice": [
        {
            "ListenAddr": "127.0.0.1:31433",
            "TargetAddr": "127.0.0.1:1433"
        }
        //socket连接到CLI(31433),CLI就转发socket到1433.
    ],
    "ForwardSlice": [
        {
            "Password": "PWD",
            "ListenAddr": "127.0.0.1:13306",
            "ConnectAddr": "localhost:10254",
            "TargetAddr": "localhost:3306"
        }
        //socket连到CLI(13306),CLI就连到SRV(10254),并让SRV转发socket到3306.
    ],
    "ReverseSlice": [
        {
            "Password": "PWD",
            "ConnectAddr": "localhost:10254",
            "SrvLisAddr": "localhost:21521",
            "TargetAddr": "127.0.0.1:1521"
        }
        //CLI连到SRV(10254),并让SRV监听21521;socket连到21521,CLI就接管socket,并转发socket到1521.
    ]
}`
	return content
}

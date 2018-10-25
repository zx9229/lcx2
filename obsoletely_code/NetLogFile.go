package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

//NetLogFile 网络通信的日志文件
//它可以记录socket发送了哪些数据&接收了哪些数据, 并为这些数据附上时间戳.
//因为发送&接收是可以同时进行的, 所以我借用chan对它们强制串行化.
//因为某些异常(比如硬盘满了之类的)导致写文件失败的话, 是无法即时反馈给Write函数的.
//一旦出现异常, 它就不可信了.
type NetLogFile struct {
	osFile   *os.File         //真正的文件句柄
	itemChan chan *NetLogItem //强制串行之用
	syncOnce sync.Once        //关闭chan之用
	errState error            //这个类是否处于错误的状态
}

func newNetLogFile(name string) (nlf *NetLogFile, err error) {
	for range "1" {
		nlf = new(NetLogFile)
		if nlf.osFile, err = os.OpenFile(name, os.O_CREATE|os.O_APPEND, 0666); err != nil {
			nlf = nil
			break
		}
		nlf.itemChan = make(chan *NetLogItem, 2048)
		go nlf.writeData()
	}
	return
}

//Close 仅关闭channel, 接收者协程处理完channel中的数据后, 会自行关闭文件句柄.
func (thls *NetLogFile) Close() {
	thls.syncOnce.Do(func() { close(thls.itemChan) })
}

//GetChildFileA2B omit
func (thls *NetLogFile) GetChildFileA2B() *ChildFileA2B {
	return &ChildFileA2B{parent: thls}
}

//GetChildFileB2A omit
func (thls *NetLogFile) GetChildFileB2A() *ChildFileB2A {
	return &ChildFileB2A{parent: thls}
}

func (thls *NetLogFile) writeData() {
	var item *NetLogItem
	var isOk bool
	var num int
	var err error
	for {
		if item, isOk = <-thls.itemChan; isOk {
			sliceSize := []byte(fmt.Sprintf("%010d", len(item.data)))
			sliceTime := []byte(item.dttm.Format("20060102 150405.000000"))
			sliceABBA := item.abba
			totalData := sliceABBA
			totalData = append(totalData, ',')
			totalData = append(totalData, sliceTime...)
			totalData = append(totalData, ',')
			totalData = append(totalData, sliceSize...)
			totalData = append(totalData, ',')
			totalData = append(totalData, item.data...)
			totalData = append(totalData, '\n')
			if num, err = thls.osFile.Write(totalData); err != nil || num != len(totalData) {
				thls.errState = fmt.Errorf("osFile.Write, want=%v, actually=%v, err=%v", len(totalData), num, err)
				log.Println(thls.errState)
				break
			}
		} else {
			break
		}
	}
	log.Println("Close", thls.osFile.Name())
	thls.osFile.Close()
}

////////////////////////////////////////////////////////////////////////////////

//ChildFileA2B 记录[A =(socket)=> B]的子日志文件.
type ChildFileA2B struct {
	parent *NetLogFile
}

func (thls *ChildFileA2B) Write(p []byte) (n int, err error) {
	if thls.parent.errState == nil {
		thls.parent.itemChan <- newNetLogItem(p, txDirectionA2B)
		n = len(p)
	} else {
		err = thls.parent.errState
	}
	return
}

//ChildFileB2A 记录[B =(socket)=> A]的子日志文件.
type ChildFileB2A struct {
	parent *NetLogFile
}

func (thls *ChildFileB2A) Write(p []byte) (n int, err error) {
	if thls.parent.errState == nil {
		thls.parent.itemChan <- newNetLogItem(p, txDirectionB2A)
		n = len(p)
	} else {
		err = thls.parent.errState
	}
	return
}

////////////////////////////////////////////////////////////////////////////////

/*
','  => 0x2C
'\r' => 0x0D
'\n' => 0x0A
*/
var (
	txDirectionA2B = []byte("A2B") //通信方向[A=>B]
	txDirectionB2A = []byte("B2A") //通信方向[B=>A]
)

//NetLogItem omit
type NetLogItem struct {
	dttm time.Time //时间戳
	data []byte    //通信数据
	abba []byte    //通信方向
}

func newNetLogItem(p []byte, dir []byte) *NetLogItem {
	byteSlice := make([]byte, len(p))
	copy(byteSlice, p)
	return &NetLogItem{dttm: time.Now(), data: byteSlice, abba: dir}
}

////////////////////////////////////////////////////////////////////////////////

//RecommendFilename 推荐给我一个合法的文件名.
func RecommendFilename(aSide net.Addr, bSide net.Addr) string {
	filename := fmt.Sprintf("%v-[%s]-[%s].log", time.Now().Format("2006_01_02_15_04_05"), aSide.String(), bSide.String())
	fields := strings.FieldsFunc(filename, func(c rune) bool { return 0 <= strings.IndexRune(`\/:*?"<>|`, c) })
	filename = strings.Join(fields, "_")
	return filename
}

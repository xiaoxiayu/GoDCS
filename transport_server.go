// Copyright 2015 xiaoxia_yu@xxsoftware.com All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	//	"bytes"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func transport_server(port string) {
	var (
		// host   = "192.168.1.5"	//如果写locahost或127.0.0.1则只能本地访问。
		//port   = "9091"
		remote = ":" + port //此方式本地与非本地都可访问
	)

	fmt.Println("Server Running... (Ctrl-C to stop)")

	lis, err := net.Listen("tcp", remote)
	defer lis.Close()

	if err != nil {
		fmt.Println("监听端口发生错误: ", remote)
		os.Exit(-1)
	}

	for {
		conn, err := lis.Accept()
		if err != nil {
			fmt.Println("客户端连接错误: ", err.Error())
			// os.Exit(0)
			continue
		}
		//调用文件接收方法
		go receiveFile(conn)
	}
}

func receiveFile(con net.Conn) {
	var (
		res          string
		tempFileName string                    //保存临时文件名称
		data         = make([]byte, 1024*1024) //用于保存接收的数据的切片
		//		by           []byte
		//		databuf      = bytes.NewBuffer(by) //数据缓冲变量
		fileNum int //当前协程接收的数据在原文件中的位置
	)
	defer con.Close()

	//fmt.Println("新建立连接: ", con.RemoteAddr())
	j := 0 //标记接收数据的次数
	for {
		length, err := con.Read(data)
		if err != nil {

			// writeend(tempFileName, databuf.Bytes())
			//da := databuf.Bytes()
			// fmt.Println("over", fileNum, len(da))
			//fmt.Printf("客户端 %v 已断开. %2d %d \n", con.RemoteAddr(), fileNum, len(da))
			return
		}

		if 0 == j {

			res = string(data[0:8])
			if "fileover" == res {
				xienum := int(data[8])
				mergeFileName := string(data[9:length])
				mergeFileName = cfg_global.RuntimePath + "/" + cfg_global.Workspace + "/" + mergeFileName
				go mainMergeFile(xienum, mergeFileName)
				res = mergeFileName
				con.Write([]byte(res))
				return
			} else {
				fileNum = int(data[0])
				tempFileName = string(data[1:length]) + strconv.Itoa(fileNum)
				tempFileName = cfg_global.RuntimePath + "/" + cfg_global.Workspace + "/" + tempFileName
				//fmt.Println("创建临时文件：", tempFileName)
				fout, err := os.Create(tempFileName)
				if err != nil {
					fmt.Printf("CreateTempFileError0:%s", tempFileName)
					return
				}
				fout.Close()
			}
		} else {
			// databuf.Write(data[0:length])
			writeTempFileEnd(tempFileName, data[0:length])
		}

		res = strconv.Itoa(fileNum) + " Recieve OK"
		con.Write([]byte(res))
		j++
	}

}

func writeTempFileEnd(fileName string, data []byte) {
	tempFile, err := os.OpenFile(fileName, os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		// panic(err)
		fmt.Printf("WriteTemFileError:%s", err.Error())
		return
	}
	defer tempFile.Close()
	tempFile.Write(data)
}

func mainMergeFile(connumber int, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("CreateMergeFileError:%s", err.Error())
		return
	}
	defer file.Close()

	for i := 0; i < connumber; i++ {
		mergeFile(filename+strconv.Itoa(i), file)
	}

	for i := 0; i < connumber; i++ {
		os.Remove(filename + strconv.Itoa(i))
	}

}

func mergeFile(rfilename string, wfile *os.File) {
	rfile, err := os.OpenFile(rfilename, os.O_RDWR, 0666)
	defer rfile.Close()
	if err != nil {
		fmt.Printf("MergeTemFileERROR:%s", rfilename)
		return
	}

	stat, err := rfile.Stat()
	if err != nil {
		fmt.Println("MergeFileStat ERROR.")
		return
	}

	num := stat.Size()

	buf := make([]byte, 1024*1024)
	for i := 0; int64(i) < num; {
		length, err := rfile.Read(buf)
		if err != nil {
			fmt.Println("MergeFileReadERROR.")
		}
		i += length

		wfile.Write(buf[:length])
	}

}

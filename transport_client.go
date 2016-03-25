// Copyright 2015 xiaoxia_yu@foxitsoftware.com All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	//	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"runtime"
	//	"strconv"
	//	"time"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func transport_client(ip, port, fileName, storeName string) error {
	var (
		host   = ip
		remote = host + ":" + port

		mergeFileName = storeName
		coroutine     = 10
		bufsize       = 1024
	)

	host = string(host)
	remote = host + ":" + port
	//	fmt.Println(remote)
	//	fmt.Println(fileName)
	fl, err := os.OpenFile(fileName, os.O_RDWR, 0666)
	if err != nil {
		fmt.Println("userFile", err)
		return err
	}

	stat, err := fl.Stat()
	if err != nil {
		return err
	}
	var size int64
	size = stat.Size()
	fl.Close()

	littleSize := size / int64(coroutine)

	c := make(chan string)
	var begin int64 = 0
	if size < 1025 {
		coroutine = 1
	}
	for i := 0; i < coroutine; i++ {

		if i == coroutine-1 {
			go splitFile(remote, c, i, bufsize, fileName, mergeFileName, begin, size)
			//fmt.Println(begin, size, bufsize)
		} else {
			go splitFile(remote, c, i, bufsize, fileName, mergeFileName, begin, begin+littleSize)
			//fmt.Println(begin, begin+littleSize)
		}

		begin += littleSize
	}

	for j := 0; j < coroutine; j++ {
		fmt.Print(<-c)
	}

	sendMergeCommand(remote, mergeFileName, coroutine) //发送文件合并指令及文件名
	return nil
}

func splitFile(remote string, c chan string, coroutineNum int, size int, fileName, mergeFileName string, begin int64, end int64) {

	con, err := net.Dial("tcp", remote)
	defer con.Close()
	if err != nil {
		fmt.Println("服务器连接失败.")
		os.Exit(-1)
		return
	}
	//fmt.Println(coroutineNum, "Link.Sending...")

	var by [1]byte
	by[0] = byte(coroutineNum)
	var bys []byte
	databuf := bytes.NewBuffer(bys)
	databuf.Write(by[:])
	databuf.WriteString(mergeFileName)
	bb := databuf.Bytes()
	// bb := by[:]
	//fmt.Println(bb)
	in, err := con.Write(bb)
	if err != nil {
		fmt.Printf("Send data to server error: %d\n", in)
		os.Exit(0)
	}

	var msg = make([]byte, 1024)
	lengthh, err := con.Read(msg)
	if err != nil {
		fmt.Printf("Read data from server error0:%s, Length:%d\n", err.Error(), lengthh)
		os.Exit(0)
	}

	file, err := os.OpenFile(fileName, os.O_RDWR, 0666)
	defer file.Close()
	if err != nil {
		fmt.Println(fileName, "Open error.")
		os.Exit(0)
	}

	file.Seek(begin, 0)
	buf := make([]byte, size)
	var sendDtaTolNum int = 0
	for i := begin; int64(i) < end; i += int64(size) {
		length, err := file.Read(buf)
		if err != nil {
			fmt.Println("Read error", i, coroutineNum, end)
		}

		if length == size {
			if int64(i)+int64(size) >= end {
				sendDataNum, err := con.Write(buf[:size-int((int64(i)+int64(size)-end))])
				if err != nil {
					fmt.Printf("Send data to server error: %d\n", sendDataNum)
					os.Exit(0)
				}
				sendDtaTolNum += sendDataNum
			} else {
				sendDataNum, err := con.Write(buf)
				if err != nil {
					fmt.Printf("Send data to server error: %d\n", sendDataNum)
					os.Exit(0)
				}
				sendDtaTolNum += sendDataNum
			}

		} else {
			sendDataNum, err := con.Write(buf[:length])
			if err != nil {
				fmt.Printf("Send data to server error: %d\n", sendDataNum)
				os.Exit(0)
			}
			sendDtaTolNum += sendDataNum
		}

		lengths, err := con.Read(msg)
		if err != nil {
			fmt.Printf("Read data from server error1.\n", lengths)
			os.Exit(0)
		}
	}

	c <- "\r"
}

func sendMergeCommand(remote, mergeFileName string, coroutine int) {

	con, err := net.Dial("tcp", remote)
	defer con.Close()
	if err != nil {
		fmt.Println("Server Link ERROR.")
		os.Exit(-1)
		return
	}
	fmt.Println("Link Success.\nMeraging...")

	var by [1]byte
	by[0] = byte(coroutine)
	var bys []byte
	databuf := bytes.NewBuffer(bys)
	databuf.WriteString("fileover")
	databuf.Write(by[:])
	databuf.WriteString(mergeFileName)
	cmm := databuf.Bytes()

	in, err := con.Write(cmm)
	if err != nil {
		fmt.Printf("Send data to server error: %d\n", in)
	}

	var msg = make([]byte, 1024)
	lengthh, err := con.Read(msg)
	if err != nil {
		fmt.Printf("Read server data error.\n", lengthh)
		os.Exit(0)
	}
	str := string(msg[0:lengthh])
	fmt.Printf("Transport Finish:%s", str)
}

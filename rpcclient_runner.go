// Copyright 2015 xiaoxia_yu@foxitsoftware.com All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"strings"
	//	"log"
	"net/rpc"
	"os"
	"path/filepath"
	//	"runtime"
	//"io"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type RunState struct {
	RunnerPort    string
	TransportPort string
	MonitorPort   string
	Ip            string
	Task          string

	Result string
}

type Run struct {
	RemoteIP      []RemoteIP
	PrjName       string
	PrjPackage    string
	ResultPackage string

	Report       []string
	ReportFlag   []string
	RecieveSpace string

	ScriptRun     string
	ScriptMonitor string
	ScriptCleaner string

	HttpPort       string
	TaskMode       string
	Task           []string
	ExternMonitorP []string
}

type RPCClientRunner struct {
	RpcPort       string
	TransportPort string
	MonitorPort   string
	args          RunnerArgs
}

func EndProject() {

}
func LinkServer2(Ip, RunnerPort, TransportPort, zipPath string, cfg Run, ch chan<- RunState) {
	clientRunner := RPCClientRunner{}
	clientRunner.LinkServerInit(Ip, RunnerPort, TransportPort, zipPath, cfg, ch)
}

func Run2(Ip, RunnerPort, TransportPort, task string, task_id int, wg *sync.WaitGroup, cfg Run, ch chan<- RunState) {
	clientRunner := RPCClientRunner{}
	clientRunner.Run(Ip, RunnerPort, TransportPort, task, task_id, wg, cfg, ch)
}

func (t *RPCClientRunner) LinkServerInit(Ip, RunnerPort, TransportPort, zipPath string, cfg Run, ch chan<- RunState) {
	if RunnerPort == "" {
		RunnerPort = "9092"
	}
	if TransportPort == "" {
		TransportPort = "9091"
	}

	fmt.Println("== Runner Try To Link: " + Ip + ":" + RunnerPort + " ==")

	stateInfo := RunState{
		Ip:            Ip,
		RunnerPort:    RunnerPort,
		TransportPort: TransportPort,
		Result:        "ERROR"}

	defer func(stateInfo *RunState, ch chan<- RunState) {
		ch <- *stateInfo
		fmt.Printf("== RunnerEnvInit %s: %s:%s ==\n", stateInfo.Result, Ip, RunnerPort)
	}(&stateInfo, ch)

	client, err := rpc.Dial("tcp", Ip+":"+RunnerPort)
	if err != nil {
		fmt.Println(err.Error())
		stateInfo.Result = "ERRORLINK"
		return
	}
	fmt.Println("== Runner Link Success: " + Ip + ":" + RunnerPort + " ==")

	t.args = RunnerArgs{}
	var reply int

	t.args.PrjName = cfg.PrjName
	t.args.Ip = Ip

	err = client.Call("Runner.Init", t.args, &reply)
	if err != nil {
		fmt.Println("LinkInitError:", err.Error())
		return
	}
	if reply == 2 {
		fmt.Printf("== Project %s already running in %s ==\n", cfg.PrjName, Ip)
		stateInfo.Result = "AlreadyRunning"
		return
	}

	//PrjPackageDir := filepath.Base(cfg.PrjPackage)
	//zipName := PrjPackageDir + ".zip"

	fmt.Printf("== Transport Start:%s ==\n", Ip)
	t.args.PrjZipPath = cfg.PrjName + "/" + filepath.Base(zipPath)

	//fmt.Println("********ZIP STORE PATH:", t.args.PrjZipPath)
	trans_err := transport_client(Ip, TransportPort, zipPath, t.args.PrjZipPath)
	if trans_err != nil {
		fmt.Printf("== Transport ERROR:%s ==\n", Ip)
		stateInfo.Result = "ERRORLINK"
		return
	} else {
		fmt.Printf("== Transport End:%s ==\n", Ip)
	}

	zip_package_status := func(args RunnerArgs) int {
		zip_package_status := 0
		//fmt.Printf("== ZipCheck Start:%s ==\n", Ip)
		for true {
			var status_reply int
			err = client.Call("Runner.CheckZipPackage", args, &status_reply)
			if err != nil {
				zip_package_status = -1
				break
			}
			if status_reply == 0 {
				time.Sleep(time.Second * 5)
				fmt.Printf("== ZipCheck Failed Retry:%s ==\n", Ip)
			} else {
				zip_package_status = 1
				fmt.Printf("== ZipCheck OK:%s ==\n", Ip)
				break
			}
		}
		return zip_package_status
	}(t.args)

	if zip_package_status == -1 {
		stateInfo.Result = "ERRORLINK"
		return
	} else {
		fmt.Printf("== ZipPackage OK:%s ==\n", Ip)
	}

	err = client.Call("Runner.UnzipPackage", t.args, &reply)
	if err != nil {
		fmt.Println("UnzipPackageERROR:", err.Error())
		return
	}
	stateInfo.Result = "Success"
}

func (t *RPCClientRunner) Run(Ip, RunnerPort, TransportPort, task string,
	task_id int, wg *sync.WaitGroup, cfg Run, ch chan<- RunState) {

	fmt.Printf("== Task%d Start:%s %s ==\n",
		task_id, Ip, task)

	stateInfo := RunState{
		Ip:            Ip,
		RunnerPort:    RunnerPort,
		TransportPort: TransportPort,
		Task:          task,
		Result:        "ERROR"}

	defer func(wg *sync.WaitGroup, stateInfo *RunState, ch chan<- RunState) {
		if cfg.TaskMode != "uniform" {
			ch <- *stateInfo
		}
		fmt.Printf("== Task%d %s:%s ==\n",
			task_id, stateInfo.Result, Ip)
		wg.Done()
		//fmt.Println("********TASK ID END*****", task_id)
	}(wg, &stateInfo, ch)

	client, err := rpc.Dial("tcp", Ip+":"+RunnerPort)
	if err != nil {
		fmt.Println("TCP:", err.Error())
		return
	}
	//fmt.Println("Link SUCCESS")
	var reply int
	args := RunnerArgs{}

	args.PrjName = cfg.PrjName
	args.Ip = Ip

	args.ScriptRun = cfg.ScriptRun
	PrjPackageDir := filepath.Base(cfg.PrjPackage)
	args.PrjPackageDir = PrjPackageDir
	args.Task = strings.Replace(task, "\"", "", -1)
	args.ExternMonitorP = cfg.ExternMonitorP

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGHUP,
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGQUIT)

		s := <-c
		fmt.Println("Task End:", s)

		args.ScriptCleaner = cfg.ScriptCleaner
		err = client.Call("Runner.KillRun", args, &reply)
		if err != nil {
			fmt.Println("KillRun:", err.Error())
		}
		err = client.Call("Runner.End", args, &reply)
		if err != nil {
			fmt.Println("End:", err.Error())
		}
		os.Exit(0)
	}()

	//fmt.Println("RUN START")

	err = client.Call("Runner.Run", args, &reply)
	if err != nil {
		fmt.Println("RUN:", err.Error())
		return
	}
	//fmt.Println("RUN END")

	args.ReportFiles = cfg.Report
	args.ReportFlags = cfg.ReportFlag
	args.TaskID = strconv.Itoa(task_id)
	var rptZipPath string
	err = client.Call("Runner.GetReport", args, &rptZipPath)
	if err != nil {
		fmt.Println("GetReport:", err.Error())
		return
	}
	//fmt.Printf("%s\n", rptZipPath)

	//fmt.Println("RUN START")
	args.ReportZipPath = rptZipPath
	var revZipBuf []byte
	err = client.Call("Runner.ReceiveReport", args, &revZipBuf)
	if err != nil {
		fmt.Println("ReceiveReport:", err.Error())
		stateInfo.Result = "RECIEVEERROR"
		return
	}

	//fmt.Println("RECEVIREPORT FINISH")
	if revZipBuf == nil {
		fmt.Printf("== Task%d %s:ReceiveReport is NULL ==\n",
			task_id, Ip)
		stateInfo.Result = "Success"
		return
	}

	revZipPath := cfg.RecieveSpace + "/" + args.Ip + "_" + args.PrjName + "_" + args.TaskID + ".zip"
	revZip, err := os.Create(revZipPath)
	if err != nil {
		stateInfo.Result = "RECIEVEERROR"
		fmt.Println("Create:", err.Error())
		return
	}

	_, err = revZip.Write(revZipBuf)
	if err != nil {
		fmt.Println("Write:", err.Error())
		stateInfo.Result = "RECIEVEERROR"
		return
	}
	revZip.Close()

	zipdestFolder := strings.Split(revZipPath, ".zip")[0]
	if Exist(zipdestFolder) {
		os.RemoveAll(zipdestFolder)
	}

	if UnZip(revZipPath) {
		os.Remove(revZipPath)
	}
	stateInfo.Result = "Success"
}

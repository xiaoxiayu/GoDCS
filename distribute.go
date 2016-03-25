// Copyright 2015 xiaoxia_yu@foxitsoftware.com All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/rpc"
	"os"
	"strings"
	"time"
	//"time"
	"path/filepath"
	//	"html/template"
	//"os/exec"
	"runtime"
	"sync"
)

type Global struct {
	RemoteIP        []RemoteIP
	Workspace       string
	TransportPort   string
	RunnerPort      string
	TaskMonitorPort string
	RuntimePath     string
}

type RemoteIP struct {
	RunnerPort      string `xml:"runnerport,attr"`
	TransportPort   string `xml:"transportport,attr"`
	TaskMonitorPort string `xml:"taskmonitorport,attr"`
	Value           string `xml:",chardata"`
}

type ServerRunningInfo struct {
	PrjName        string
	ScriptName     string
	ScriptPID      int
	StartTime      string
	ExternMonitorP []string
}

var cfg_global Global
var runninginfo_global ServerRunningInfo

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func RunDistributive(cfg Run) {
	fmt.Println("TaskMode:Distributive, Remote machine count:", len(cfg.RemoteIP))

	go StartHttpMonitor(cfg)

	PrjPackageDir := filepath.Base(cfg.PrjPackage)
	zipPath := cfg_global.RuntimePath + "/" + cfg.PrjName + "_" + PrjPackageDir + ".zip"
	ZipFolder(cfg.PrjPackage, zipPath)

	defer func() {
		if Exist(zipPath) {
			os.Remove(zipPath)
		}
		fmt.Println("== END. ==")
	}()

	chs := make(chan RunState, len(cfg.RemoteIP))

	var task string
	var task_id int
	var err_i int
	var wg sync.WaitGroup
	for remote_id, Ip := range cfg.RemoteIP {
		go LinkServer2(Ip.Value, Ip.RunnerPort, Ip.TransportPort, zipPath, cfg, chs)
		//fmt.Println("TASK ID:", task_id)
		if remote_id+1 == len(cfg.RemoteIP) {
			for {
				fmt.Println("START NEW TASK")
				result, _ := <-chs
				if task_id >= len(cfg.Task) {
					fmt.Println("== All the tasks are starting to run ==")
					break
				}

				if strings.Contains(result.Result, "Success") {
					if task_id >= len(cfg.Task) {
						continue
					}
					task = cfg.Task[task_id]
					wg.Add(1)
					go Run2(result.Ip, result.RunnerPort, result.TransportPort, task, task_id, &wg, cfg, chs)
					task_id++
					err_i = 0
				} else if strings.Contains(result.Result, "AlreadyRunning") {
					time.Sleep(30 * time.Second)
					fmt.Printf("== Task%d %s: ReLink ==", task_id, result.Ip)

					go LinkServer2(result.Ip, result.RunnerPort, result.TransportPort, zipPath, cfg, chs)
				} else if strings.Contains(result.Result, "ERRORLINK") {
					fmt.Printf("== Task%d %s: LinkERROR ==", task_id, result.Ip)
					time.Sleep(15 * time.Second)
					go LinkServer2(result.Ip, result.RunnerPort, result.TransportPort, zipPath, cfg, chs)
				} else if strings.Contains(result.Result, "RECIEVEERROR") {
					fmt.Printf("== Task%d %s: ReceiveReportERROR ==", task_id, result.Ip)
					if task_id >= len(cfg.Task) {
						continue
					}
					task = cfg.Task[task_id]
					wg.Add(1)
					go Run2(result.Ip, result.RunnerPort, result.TransportPort, task, task_id, &wg, cfg, chs)
					task_id++
					err_i = 0

				} else {
					err_i++
					fmt.Printf("== Task%d %s: ERROR:%s ==", task_id, result.Ip, result.Result)

					if err_i >= ( /*2 **/ len(cfg.RemoteIP)) {
						fmt.Println("== ALL CONNECT ERROR ==")
						log.Fatal("== ALL CONNET ERROR ==")
					}
				}
			}
			break
		}
	}
	wg.Wait()

}

func RunUniform(cfg Run) {
	fmt.Println("TaskMode:Uniform, Remote machine count:", len(cfg.RemoteIP))

	go StartHttpMonitor(cfg)

	PrjPackageDir := filepath.Base(cfg.PrjPackage)
	zipPath := cfg_global.RuntimePath + "/" + cfg.PrjName + "_" + PrjPackageDir + ".zip"
	ZipFolder(cfg.PrjPackage, zipPath)

	defer func() {
		if Exist(zipPath) {
			os.Remove(zipPath)
		}
		fmt.Println("== END. ==")
	}()

	chs := make(chan RunState, len(cfg.RemoteIP))

	//	var task string
	var err_i int
	var wg sync.WaitGroup
	for task_id, task := range cfg.Task {
		for _, Ip := range cfg.RemoteIP {
			go LinkServer2(Ip.Value, Ip.RunnerPort, Ip.TransportPort, zipPath, cfg, chs)
			result, _ := <-chs
			if strings.Contains(result.Result, "Success") {
				wg.Add(1)
				go Run2(result.Ip, result.RunnerPort, result.TransportPort, task, task_id, &wg, cfg, chs)
				err_i = 0
			} else if strings.Contains(result.Result, "AlreadyRunning") {
				time.Sleep(30 * time.Second)
				fmt.Printf("== Task%d %s: ReLink ==", task_id, result.Ip)

				go LinkServer2(result.Ip, result.RunnerPort, result.TransportPort, zipPath, cfg, chs)
			} else if strings.Contains(result.Result, "ERRORLINK") {
				fmt.Printf("== Task%d %s: LinkERROR ==", task_id, result.Ip)
				time.Sleep(15 * time.Second)
				go LinkServer2(result.Ip, result.RunnerPort, result.TransportPort, zipPath, cfg, chs)
			} else if strings.Contains(result.Result, "RECIEVEERROR") {
				fmt.Printf("== Task%d %s: ReceiveReportERROR ==", task_id, result.Ip)
				wg.Add(1)
				go Run2(result.Ip, result.RunnerPort, result.TransportPort, task, task_id, &wg, cfg, chs)
				err_i = 0

			} else {
				err_i++
				fmt.Printf("== Task%d %s: ERROR:%s ==", task_id, result.Ip, result.Result)

				if err_i >= ( /*2 **/ len(cfg.RemoteIP)) {
					fmt.Println("== ALL CONNECT ERROR ==")
					log.Fatal("== ALL CONNET ERROR ==")
				}
			}
		}
	}

	wg.Wait()
}

func RunDCS() {
	cfg_global.RuntimePath = GetRuntimePath()
	fmt.Println("Start:" + cfg_global.RuntimePath)
	os.Chdir(cfg_global.RuntimePath)
	if len(os.Args) > 1 {

		var cfg Run
		xml_file := os.Args[1]
		fmt.Print(xml_file + "\n")
		file, err := ioutil.ReadFile(xml_file)
		if err != nil {
			fmt.Printf("%s Read ERROR: %v\n", xml_file, err)
		}

		err = xml.Unmarshal(file, &cfg)
		if err != nil {
			fmt.Printf("%s XML Parse ERROR: %v\n", xml_file, err)
		}

		if !Exist(cfg.RecieveSpace) {
			os.Mkdir(cfg.RecieveSpace, os.ModePerm)
		}

		if cfg.TaskMode == "uniform" {
			RunUniform(cfg)
		} else {
			RunDistributive(cfg)
		}

		return
	} else {

		xml_file := "globalcfg.xml"
		//fmt.Print(xml_file + "\n")
		file, err := ioutil.ReadFile(xml_file)
		if err != nil {
			fmt.Printf("%s Read ERROR: %v\n", xml_file, err)
		}
		//cfg_global = Global{}
		err = xml.Unmarshal(file, &cfg_global)
		if err != nil {
			fmt.Printf("%s XML Parse ERROR: %v\n", xml_file, err)
		}
		//fmt.Println(cfg_global)
		if !Exist(cfg_global.Workspace) {
			os.Mkdir(cfg_global.Workspace, os.ModePerm)
		}

		rpc.Register(NewRPC())

		monitor_l, monitor_e := net.Listen("tcp", ":"+cfg_global.TaskMonitorPort)
		if monitor_e != nil {
			log.Fatal("RPCMonitor error:", monitor_e)
		}

		go rpc.Accept(monitor_l)
		fmt.Printf("== TaskMonitor Start: :%s ==\n", cfg_global.TaskMonitorPort)

		runner := new(Runner)
		runner.RunningPrj = make(map[string]bool)
		runner.Locker = new(sync.Mutex)
		rpc.Register(runner)
		rpc.HandleHTTP()

		l, e := net.Listen("tcp", ":"+cfg_global.RunnerPort)
		if e != nil {
			log.Fatal("RPCData listen error:", e)
		}

		go transport_server(cfg_global.TransportPort)
		fmt.Printf("== RPCTransporter Start: :%s ==\n", cfg_global.TransportPort)

		rpc.Accept(l)
	}
}

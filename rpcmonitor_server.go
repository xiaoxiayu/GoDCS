// Copyright 2015 xiaoxia_yu@xxsoftware.com All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	//	"time"
	//	"path/filepath"
	//	"bytes"
	//	"io"
	//	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type (
	RPC struct {
		cache    map[string]string
		requests *Requests
		mu       *sync.RWMutex
	}

	CacheItem struct {
		Key   string
		Value string
	}

	MonitorItem struct {
		Script string
		Pid    string
	}

	MonitorData struct {
		OutInfo *[]byte
		ErrInfo *[]byte
	}

	Requests struct {
		GetStateInfo uint64
	}
)

func NewRPC() *RPC {
	return &RPC{
		cache:    make(map[string]string),
		requests: &Requests{},
		mu:       &sync.RWMutex{},
	}
}

func (r *RPC) GetStateInfo(args *MonitorItem, resp *[]byte) (err error) {
	workspacePath := cfg_global.RuntimePath + "/" + cfg_global.Workspace
	scriptPath := workspacePath + "/" + args.Script
	if runtime.GOOS == "windows" {
		var cmd_str string
		if path.Ext(args.Script) == ".py" {
			cmd_str = "python " + scriptPath
		} else {
			cmd_str = "cmd.exe /c call " + scriptPath
		}

		fmt.Println("MONITOR SCRIPT:", cmd_str)
		cmd_list := strings.Split(cmd_str, " ")
		out, err := exec.Command(cmd_list[0], cmd_list[1:]...).CombinedOutput()
		if err != nil {
			errStr := "Run Monitor script error:" + err.Error()
			fmt.Println(errStr)
			*resp = []byte(errStr)
			return nil
		}
		*resp = out
	} else {
		cmd := exec.Command("chmod", "+x", "./"+scriptPath)
		//		cmd.Stdout = os.Stdout
		//		cmd.Stderr = os.Stderr
		cmd.Start()
		cmd.Run()
		cmd.Wait()

		cmd_str := "./" + scriptPath
		out, err := exec.Command(cmd_str).CombinedOutput()
		if err != nil {
			errStr := "Run Monitor script error:" + err.Error()
			fmt.Println(errStr)
			*resp = []byte(errStr)
			return nil
		}
		fmt.Println(out)
		*resp = out
	}
	return nil
}

func (r *RPC) GetRunningInfo(args *MonitorItem, resp *[]byte) (err error) {
	info := "\t" + runninginfo_global.PrjName + "\t" + runninginfo_global.StartTime
	info += "\t" + runninginfo_global.ScriptName + "\t" + strconv.Itoa(runninginfo_global.ScriptPID)
	info += "_Platform_" + runtime.GOOS
	info += "_ExternMonitorP_"

	if runtime.GOOS == "windows" {
		for _, proc := range runninginfo_global.ExternMonitorP {
			out, err := exec.Command("tasklist.exe", "/FI", "IMAGENAME eq "+proc).CombinedOutput()
			if err != nil {
				errStr := "Run Monitor script error:" + err.Error()
				fmt.Println(errStr)
			}
			info += string(out)
		}

	} else {
		for _, proc := range runninginfo_global.ExternMonitorP {
			out, err := exec.Command("pgrep", "-f", "-l", proc).CombinedOutput()
			if err != nil {
				errStr := "Run Monitor script error:" + err.Error()
				fmt.Println(errStr)
			}
			fmt.Println(string(out))
			info += string(string(out))
		}
	}

	*resp = []byte(info)
	return nil
}

func (r *RPC) KillProcess(args *MonitorItem, state *int) error {
	fmt.Println("Kill Process:", args.Pid)
	if runtime.GOOS == "windows" {

		_, err := exec.Command("taskkill.exe", "/F", "/PID", args.Pid).CombinedOutput()
		if err != nil {
			errStr := "Kill error:" + err.Error()
			fmt.Println(errStr)
		}
		//			info += string(out)

	} else {

		_, err := exec.Command("kill", "-9", args.Pid).CombinedOutput()
		if err != nil {
			errStr := "Kill error:" + err.Error()
			fmt.Println(errStr)
		}
		//	info += string(out)

	}
	*state = 1
	return nil
}

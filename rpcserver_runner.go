// Copyright 2015 xiaoxia_yu@foxitsoftware.com All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type RunnerArgs struct {
	PrjZipPath string
	PrjName    string

	ScriptRun     string
	ScriptCleaner string

	PrjPackageDir string
	ReportFiles   []string
	ReportFlags   []string
	Ip            string
	ReportZipPath string
	Task          string
	TaskID        string

	ExternMonitorP []string
}

type Runner struct {
	Pid        int
	Locker     *sync.Mutex
	RunningPrj map[string]bool
}

func Try(fun func(), handler func(interface{})) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("**Have a panic:%v**", err)

			handler(err)

		}
	}()
	fun()
}

func SetPrjPath(args *RunnerArgs) string {
	exePath, _ := exec.LookPath(os.Args[0])
	exePath, _ = filepath.Abs(exePath)
	workPath := filepath.Dir(exePath)

	rpj_path := cfg_global.RuntimePath + "/" + cfg_global.Workspace + "/" + args.PrjName + "/" + args.PrjName + "_" + args.PrjPackageDir

	os.Chdir(rpj_path)
	fmt.Println("**SET WORKPATH:", rpj_path)
	return workPath
}

func (t *Runner) Init(args *RunnerArgs, state *int) error {
	//os.Chdir(cfg_global.RuntimePath)
	_, isExist := t.RunningPrj[args.PrjName]
	if isExist {
		*state = 2
		return nil
	}

	prjPath := cfg_global.Workspace + "/" + args.PrjName
	if Exist(prjPath) {
		os.RemoveAll(prjPath)
	}
	os.Mkdir(prjPath, os.ModePerm)
	//	t.RunningPrj = append(t.RunningPrj, args.PrjName)

	*state = 1
	return nil
}

func (t *Runner) UnzipPackage(args *RunnerArgs, state *int) error {
	zipPath := cfg_global.RuntimePath + "/" + cfg_global.Workspace + "/" + args.PrjZipPath
	ret := UnZip(zipPath)
	if ret {
		os.Remove(zipPath)
	} else {
		fmt.Println("UnzipPackageError")
	}
	*state = 1
	return nil
}

func (t *Runner) CheckZipPackage(args *RunnerArgs, state *int) error {
	zipPath := cfg_global.RuntimePath + "/" + cfg_global.Workspace + "/" + args.PrjZipPath
	err := CheckZipFile(zipPath)
	if err == nil {
		*state = 1
	} else {
		*state = 0
	}
	return nil
}

func (t *Runner) Run(args *RunnerArgs, state *int) error {
	var cmd_str string
	var cmd *exec.Cmd
	var workPath string

	t.Locker.Lock()
	defer func() {
		runninginfo_global = ServerRunningInfo{}
		runninginfo_global.ExternMonitorP = args.ExternMonitorP
		os.Chdir(workPath)
		t.Locker.Unlock()
		delete(t.RunningPrj, args.PrjName)
	}()
	//
	runninginfo_global.ScriptName = args.ScriptRun
	runninginfo_global.PrjName = args.PrjName
	runninginfo_global.StartTime = time.Now().Format("2006-01-02 15:04:05")
	runninginfo_global.ExternMonitorP = args.ExternMonitorP

	t.RunningPrj[args.PrjName] = true

	if runtime.GOOS == "windows" {

		if path.Ext(args.ScriptRun) == ".py" {
			cmd_str = "python " + args.ScriptRun + " " + args.Task
		} else {
			cmd_str = "cmd.exe /c call " + args.ScriptRun + " " + args.Task
		}

		cmd_list := strings.Split(cmd_str, " ")
		workPath = SetPrjPath(args)
		fmt.Printf("== %s RUN Start:%s ==\n", args.Ip, cmd_str)
		cmd = exec.Command(cmd_list[0], cmd_list[1:]...)
		//cmd := exec.Command(cmd_str)
		//		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		t.Pid = -1
		err := cmd.Start()
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				t.Pid = -1
			} else {
				t.Pid = cmd.Process.Pid
			}
		} else {
			t.Pid = cmd.Process.Pid
		}
		runninginfo_global.ScriptPID = t.Pid
		fmt.Println("RUN Process PID: ", t.Pid)

		cmd.Run()
	} else {
		workPath = SetPrjPath(args)
		rpj_path := cfg_global.RuntimePath + "/" + cfg_global.Workspace +
			"/" + args.PrjName + "/" + args.PrjName + "_" + args.PrjPackageDir

		script_path := rpj_path + "/" + args.ScriptRun
		cmd = exec.Command("chmod", "+x", script_path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Start()
		cmd.Run()
		cmd.Wait()

		cmd_str = script_path + " " + args.Task //"./" + args.ScriptRun + " " + args.Task
		//		cmd_list := strings.Split(cmd_str, " ")
		fmt.Printf("== RUN:%s ==\n", cmd_str)
		cmd = exec.Command("bash", "-c", cmd_str)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Start()
		if err != nil {
			fmt.Println("CMD START ERROR:", err.Error())
		}
		t.Pid = cmd.Process.Pid
		runninginfo_global.ScriptPID = cmd.Process.Pid
		fmt.Println("RUN Process PID: ", t.Pid)
		cmd.Run()

	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		cmd.Wait()
		wg.Done()
	}(&wg)

	wg.Wait()

	*state = 1
	fmt.Printf("== %s RUN End ==\n", args.Ip)

	return nil
}

func (t *Runner) GetReport(args *RunnerArgs, rptZipPath *string) error {
	prjPath := cfg_global.RuntimePath + "/" + cfg_global.Workspace + "/" + args.PrjName + "/" + args.PrjName + "_" + args.PrjPackageDir
	rptPath := prjPath + "/" + args.PrjName + "_report_" + args.TaskID

	//fmt.Println("*** REPORT PATH:", rptPath)
	if Exist(rptPath) {
		os.RemoveAll(rptPath)
	}

	os.Mkdir(rptPath, os.ModePerm)
	if len(args.ReportFiles) == 0 {
		return nil
	}

	for _, reportFile := range args.ReportFiles {
		fileAbsPath := prjPath + "/" + reportFile
		if IsDir(fileAbsPath) {
			CopyFolder(fileAbsPath, rptPath+"/"+reportFile, args.ReportFlags)
		} else {
			CopyFile(fileAbsPath, rptPath+"/"+reportFile)
		}
	}

	zipPath := prjPath + "/" + args.Ip + "_" + args.PrjName + "_report_" + args.TaskID + ".zip"
	//fmt.Println("*** REPORT zipPath:", zipPath)
	ZipFolder(rptPath, zipPath)

	*rptZipPath = zipPath
	return nil
}

func (t *Runner) ReceiveReport(args *RunnerArgs, zipBuf *[]byte) error {
	if !Exist(args.ReportZipPath) {
		fmt.Println("==NOT FOUND==:" + args.ReportZipPath)
		//		os.Chdir(workPath)
		return nil
	}
	fileSize := GetFileSize(args.ReportZipPath)
	if fileSize > 200000000 {
		sizeLimitStr := "Report is too large. Limit Size: 200M."
		*zipBuf = []byte(sizeLimitStr)
		return nil
	}
	fin, err := os.Open(args.ReportZipPath)
	if err != nil {
		fmt.Printf("ReceiveRportERROR:%s,%s", args.ReportZipPath, err.Error())
		zipBuf = nil
		return nil
	}
	buf := make([]byte, 1024)
	for {
		n, _ := fin.Read(buf)
		if 0 == n {
			break
		}
		//*zipBuf += buf
		*zipBuf = append(*zipBuf, buf[:n]...)
	}
	fin.Close()
	os.Remove(args.ReportZipPath)

	delete(t.RunningPrj, args.PrjName)

	return nil
}

func (t *Runner) KillRun(args *RunnerArgs, state *int) error {
	if t.Pid == 0 {
		return nil
	}
	fmt.Println("KillRun Process: ", t.Pid)
	p, err := os.FindProcess(t.Pid)
	if err == nil {
		p.Kill()
	}

	defer func() {
		runninginfo_global = ServerRunningInfo{}
	}()

	var cmd_str string
	var cmd *exec.Cmd
	var workPath string
	if runtime.GOOS == "windows" {
		if path.Ext(args.ScriptRun) == ".py" {
			cmd_str = "python " + args.ScriptCleaner + " " + args.Task
		} else {
			cmd_str = "cmd.exe /c call " + args.ScriptCleaner + " " + args.Task
		}

		cmd_list := strings.Split(cmd_str, " ")
		fmt.Printf("== %s Cleaner Start:%s ==\n", args.Ip, cmd_str)
		cmd = exec.Command(cmd_list[0], cmd_list[1:]...)
		//cmd := exec.Command(cmd_str)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		workPath = SetPrjPath(args)
		cmd.Start()

		cmd.Run()

	} else {
		cmd = exec.Command("chmod", "+x", "./"+args.ScriptCleaner)
		//		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Start()
		cmd.Run()
		cmd.Wait()

		cmd_str = "./" + args.ScriptCleaner + " " + args.Task
		fmt.Printf("== Cleaner:%s ==\n", cmd_str)
		cmd = exec.Command(cmd_str)
		//cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		workPath = SetPrjPath(args)
		cmd.Start()

		cmd.Run()
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		cmd.Wait()
		wg.Done()
	}(&wg)
	os.Chdir(workPath)

	wg.Wait()

	*state = 1
	//fmt.Printf("== %s Cleaner End ==\n", args.Ip)

	return nil
}

func (t *Runner) End(args *RunnerArgs, state *int) error {
	defer func() {
		runninginfo_global = ServerRunningInfo{}
	}()

	delete(t.RunningPrj, args.PrjName)

	*state = 1
	return nil
}

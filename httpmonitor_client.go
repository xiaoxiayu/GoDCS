// Copyright 2015 xiaoxia_yu@xxsoftware.com All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"path/filepath"
	//	"html/template"
	//	"log"
	"net/http"
	"net/rpc"
	"regexp"
	"time"
)

type HttpMonitor struct {
	ScriptMonitor string
	RpcMap        map[string]*rpc.Client
}

type home struct {
	Title string
}

var mux map[string]func(http.ResponseWriter, *http.Request)

func (t *HttpMonitor) Index(w http.ResponseWriter, r *http.Request) {
	var allInfo []byte
	for ip, rpcMonitor := range t.RpcMap {
		item := MonitorItem{Script: t.ScriptMonitor}
		var remoteInfo []byte
		rpcMonitor.Call("RPC.GetStateInfo", item, &remoteInfo)
		//fmt.Println(remoteInfo)
		allInfo = append(allInfo, []byte(ip)...)
		allInfo = append(allInfo, []byte(":\t\n\t")...)
		allInfo = append(allInfo, remoteInfo...)
	}
	fmt.Fprintf(w, string(allInfo))
	//fmt.Println("HTTP Index Run")
}

func (*HttpMonitor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h, ok := mux[r.URL.String()]; ok {
		//fmt.Println(mux)
		h(w, r)
		return
	}
	if ok, _ := regexp.MatchString("/css/", r.URL.String()); ok {
		//fmt.Println("2")
		http.StripPrefix("/css/", http.FileServer(http.Dir("./css/"))).ServeHTTP(w, r)
	} else {
		//fmt.Println("3")
		http.StripPrefix("/", http.FileServer(http.Dir("."))).ServeHTTP(w, r)
	}
}

func (t *HttpMonitor) Init(cfg Run) {
	t.ScriptMonitor = cfg.PrjName + "/" + cfg.PrjName + "_" + filepath.Base(cfg.PrjPackage) + "/" + cfg.ScriptMonitor
	t.RpcMap = make(map[string]*rpc.Client)

	for _, Ip := range cfg.RemoteIP {
		if Ip.TaskMonitorPort == "" {
			Ip.TaskMonitorPort = "9093"
		}
		fmt.Println("== Monitor Try To Link: " + Ip.Value + ":" + Ip.TaskMonitorPort + " ==\n")
		client, err := rpc.Dial("tcp", Ip.Value+":"+Ip.TaskMonitorPort)
		if err != nil {
			fmt.Println("Monitor connect error:" + err.Error())
			fmt.Println("== Monitor Link ERROR: " + Ip.Value + ":" + Ip.TaskMonitorPort + " ==")
			continue
		}
		t.RpcMap[Ip.Value+":"+Ip.TaskMonitorPort] = client

		fmt.Println("== Monitor Link Success: " + Ip.Value + ":" + Ip.TaskMonitorPort + " ==\n")
	}
}

func StartHttpMonitor(cfg Run) {
	http_serv := new(HttpMonitor)
	http_serv.Init(cfg)

	server := http.Server{
		Addr:        ":" + cfg.HttpPort,
		Handler:     &HttpMonitor{},
		ReadTimeout: 5 * time.Second,
	}

	mux = make(map[string]func(http.ResponseWriter, *http.Request))

	mux["/"] = http_serv.Index

	server.ListenAndServe()
}

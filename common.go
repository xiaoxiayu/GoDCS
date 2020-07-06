// Copyright 2015 xiaoxia_yu@xxsoftware.com All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	//	"path"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
)

func ReSetWorkPath() {
	exePath, _ := exec.LookPath(os.Args[0])
	exePath, _ = filepath.Abs(exePath)
	workPath := filepath.Dir(exePath)
	os.Chdir(workPath)
}

func CopyFolder(source string, dest string, filter []string) (err error) {

	sourceinfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dest, sourceinfo.Mode())
	if err != nil {
		return err
	}

	directory, _ := os.Open(source)

	objects, err := directory.Readdir(-1)

	for _, obj := range objects {

		sourcefilepointer := source + "/" + obj.Name()

		destinationfilepointer := dest + "/" + obj.Name()

		if obj.IsDir() {
			err = CopyFolder(sourcefilepointer, destinationfilepointer, filter)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			for _, filterFlag := range filter {
				if strings.Contains(sourcefilepointer, filterFlag) {
					err = CopyFile(sourcefilepointer, destinationfilepointer)
					if err != nil {
						fmt.Println(err)
					}
				}
			}

		}

	}
	return
}

func CopyFile(source string, dest string) (err error) {
	sourcefile, err := os.Open(source)
	if err != nil {
		return err
	}

	defer sourcefile.Close()

	dstDir := filepath.Dir(dest)
	if !Exist(dstDir) {
		os.MkdirAll(dstDir, os.ModePerm)
	}

	destfile, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer destfile.Close()

	_, err = io.Copy(destfile, sourcefile)
	if err == nil {
		sourceinfo, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(dest, sourceinfo.Mode())
		}

	}

	return
}

func GetFileSize(file string) int64 {
	f, e := os.Stat(file)
	if e != nil {
		return 0
	}
	return f.Size()
}

func GetRuntimePath() string {
	exePath, _ := exec.LookPath(os.Args[0])
	exePath, _ = filepath.Abs(exePath)
	workPath := filepath.Dir(exePath)
	return workPath
}

func SubString(str string, start, length int) (substr string) {
	rs := []rune(str)
	rl := len(rs)
	end := 0

	if start < 0 {
		start = rl - 1 + start
	}
	end = start + length

	if start > end {
		start, end = end, start
	}

	if start < 0 {
		start = 0
	}
	if start > rl {
		start = rl
	}
	if end < 0 {
		end = 0
	}
	if end > rl {
		end = rl
	}

	return string(rs[start:end])
}

func isFileOrDir(filename string, decideDir bool) bool {
	fileInfo, err := os.Stat(filename)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	isDir := fileInfo.IsDir()
	if decideDir {
		return isDir
	}
	return !isDir
}

func IsDir(filename string) bool {
	return isFileOrDir(filename, true)
}

func Exist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func GetFilelist(path_str string) []string {
	file_list := []string{}
	err := filepath.Walk(path_str, func(path_str string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		file_list = append(file_list, path_str)
		return nil
	})
	if err != nil {
		fmt.Printf("filepath.Walk() returned %v\n", err)
	}
	return file_list
}

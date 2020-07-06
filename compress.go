// Copyright 2015 xiaoxia_yu@xxsoftware.com All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	singleFileByteLimit = 107374182400 // 1 GB
	chunkSize           = 4096         // 4 KB
)

func CheckZipFile(src_zip string) error {
	unzip_file, err := zip.OpenReader(src_zip)
	if err != nil {
		return err
	}
	unzip_file.Close()
	return err
}

func UnZip(src_zip string) bool {
	dest := strings.Split(src_zip, ".zip")[0]
	unzip_file, err := zip.OpenReader(src_zip)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	os.MkdirAll(dest, 0755)
	for _, f := range unzip_file.File {
		rc, err := f.Open()
		if err != nil {
			fmt.Println(err.Error())
			return false
		}
		path := filepath.Join(dest, f.Name)
		//fmt.Println(path)
		path = strings.Replace(path, "\\", "/", -1)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			dirPath := filepath.Dir(path)
			if !Exist(dirPath) {
				os.MkdirAll(dirPath, os.ModePerm)
			}
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				fmt.Println(err.Error())
				return false
			}
			_, err = io.Copy(f, rc)
			if err != nil {
				if err != io.EOF {
					fmt.Println(err.Error())
					return false
				}
			}
			f.Close()
		}
	}
	unzip_file.Close()
	return true
}

func copyContents(r io.Reader, w io.Writer) error {
	var size int64
	b := make([]byte, chunkSize)
	for {
		// we limit the size to avoid zip bombs
		size += chunkSize
		if size > singleFileByteLimit {
			return errors.New("file too large, please contact us for assistance")
		}
		// read chunk into memory
		length, err := r.Read(b[:cap(b)])
		if err != nil {
			if err != io.EOF {
				return err
			}
			if length == 0 {
				break
			}
		}
		// write chunk to zip file
		_, err = w.Write(b[:length])
		if err != nil {
			return err
		}
	}
	return nil
}

type zipper struct {
	srcFolder string
	destFile  string
	writer    *zip.Writer
}

func (z *zipper) zipFile(path string, f os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if !f.Mode().IsRegular() || f.Size() == 0 {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var fileName string
	if runtime.GOOS == "windows" {
		fileName = strings.TrimPrefix(path, z.srcFolder+"\\")
	} else {
		fileName = strings.TrimPrefix(path, z.srcFolder+"/")
	}

	//fmt.Println("zipFile fileName:", path)
	w, err := z.writer.Create(fileName)
	if err != nil {
		return err
	}

	err = copyContents(file, w)
	if err != nil {
		return err
	}
	return nil
}

func (z *zipper) zipFolder() error {
	zipFile, err := os.Create(z.destFile)
	if err != nil {
		return err
	}
	defer zipFile.Close()
	// create zip writer
	z.writer = zip.NewWriter(zipFile)

	os.Chdir(z.srcFolder)
	defer func() {
		os.Chdir(cfg_global.RuntimePath)
	}()

	err = filepath.Walk(".", z.zipFile)
	if err != nil {
		return nil
	}
	// close the zip file
	err = z.writer.Close()
	if err != nil {
		return err
	}
	return nil
}

func ZipFolder(srcFolder string, destFile string) error {

	z := &zipper{
		srcFolder: srcFolder,
		destFile:  destFile,
	}
	return z.zipFolder()
}

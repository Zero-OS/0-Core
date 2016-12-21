package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	dir = flag.String("dir", "./tmp", "working dir")
)

func main() {
	flag.Parse()

	start := time.Now()
	err := readAllFilesInDir(*dir)
	log.Printf("readall %v, time=%v, err=%v", *dir, time.Since(start), err)
}

func readAllFilesInDir(dir string) error {
	return filepath.Walk(dir, readFileInWalk)
}

func readFileInWalk(name string, info os.FileInfo, err error) error {
	if info.IsDir() {
		return nil
	}

	f, err := os.Open(name)
	if err != nil {
		return fmt.Errorf("file=%v,err=%v", name, err)
	}
	defer f.Close()

	//log.Printf("processing:%v", name)
	if _, err = io.Copy(ioutil.Discard, f); err != nil {
		return fmt.Errorf("read file failed, file=%v,err=%v", name, err)
	}

	return nil
}

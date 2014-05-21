package main

import (
	"flag"
	"github.com/wangkuiyi/file"
	"github.com/wangkuiyi/prism"
	"log"
	"net/rpc"
	"os"
	"path"
)

var (
	addrFlag = flag.String("addr", ":12340", "The address of Prism")
)

func main() {
	flag.Parse()

	if e := file.Initialize(); e != nil {
		log.Fatalf("file.Initalize() :%v", e)
	}

	exe := path.Join(file.LocalPrefix+path.Dir(os.Args[0]), "hello")
	if _, e := file.Put(exe, "hdfs:/hello"); e != nil {
		log.Fatalf("Put %s error: %v", exe, e)
	}

	c, e := rpc.DialHTTP("tcp", *addrFlag)
	if e != nil {
		log.Fatalf("Dialing %s error: %v", *addrFlag, e)
	}

	e = c.Call("Prism.Launch",
		&prism.Deployment{"hdfs:/hello", "file:/tmp", "hello"}, nil)
	if e != nil {
		log.Fatalf("Call: %v", e)
	}
}

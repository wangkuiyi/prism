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

	log.Println("Initialize connection to HDFS ...")
	if e := file.Initialize(); e != nil {
		log.Fatalf("file.Initalize() :%v", e)
	}
	log.Println("Done")

	exe := path.Join(file.LocalPrefix+path.Dir(os.Args[0]), "hello")
	log.Println("Upload", exe, "to HDFS ...")
	if _, e := file.Put(exe, "hdfs:/hello"); e != nil {
		log.Fatalf("Put %s error: %v", exe, e)
	}
	log.Println("Done")

	log.Println("DialHTTP to Prism server ...")
	c, e := rpc.DialHTTP("tcp", *addrFlag)
	if e != nil {
		log.Fatalf("Dialing %s error: %v", *addrFlag, e)
	}
	log.Println("Done")

	if e := c.Call("Prism.DeployAndLaunchFile", &prism.DeployAndCmd{
		RemoteDir: "hdfs:/hello",
		LocalDir:  "file:/tmp",
		Filename:  "hello",
		Args:      []string{},
		LogBase:   "file:/tmp",
		Retry:     2}, nil); e != nil {
		log.Fatalf("Prism.Launch: %v", e)
	}
}

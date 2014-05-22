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
	prismFlag  = flag.String("prism", ":12340", "The address of Prism")
	actionFlag = flag.String("action", "start", "{launch, kill}")
)

func main() {
	flag.Parse()

	switch *actionFlag {
	case "launch":
		launch()
	case "kill":
		kill()
	}
}

func launch() {
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
	c, e := rpc.DialHTTP("tcp", *prismFlag)
	if e != nil {
		log.Fatalf("Dialing %s error: %v", *prismFlag, e)
	}
	defer c.Close()
	log.Println("Done")

	if e := c.Call("Prism.Deploy",
		&prism.Program{
			RemoteDir: "hdfs:/hello",
			LocalDir:  "file:/tmp",
			Filename:  "hello"}, nil); e != nil {
		log.Fatalf("Prism.Deploy failed: %v", e)
	}

	if e = c.Call("Prism.Launch",
		&prism.Cmd{
			Addr:     "localhost:8080",
			LocalDir: "file:/tmp",
			Filename: "hello",
			Args:     []string{},
			LogBase:  "file:/tmp",
			Retry:    2}, nil); e != nil {
		log.Fatalf("Prism.Launch: %v", e)
	}
}

func kill() {
	log.Println("DialHTTP to Prism server ...")
	c, e := rpc.DialHTTP("tcp", *prismFlag)
	if e != nil {
		log.Fatalf("Dialing %s error: %v", *prismFlag, e)
	}
	defer c.Close()
	log.Println("Done")

	if e = c.Call("Prism.Kill", "localhost:8080", nil); e != nil {
		log.Fatalf("Prism.Kill: %v", e)
	}
}

package main

import (
	"flag"
	"github.com/wangkuiyi/file"
	"github.com/wangkuiyi/prism"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

var (
	addrFlag = flag.String("addr", ":12340", "The listen address")
)

func main() {
	flag.Parse()
	if e := file.Initialize(); e != nil {
		log.Fatalf("file.Initalize() :%v", e)
	}

	s := prism.NewPrism()
	rpc.Register(s)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", *addrFlag)
	if e != nil {
		log.Fatalf("Cannot listen on %s: %v", *addrFlag, e)
	}
	log.Printf("Listening on %s", *addrFlag)
	http.Serve(l, nil)
}

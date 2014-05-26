package main

import (
	"flag"
	"fmt"
	"github.com/wangkuiyi/file"
	"github.com/wangkuiyi/prism"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

func main() {
	flag.Parse()
	if e := file.Initialize(); e != nil {
		log.Fatalf("file.Initalize() :%v", e)
	}

	s := prism.NewPrism()
	rpc.Register(s)
	rpc.HandleHTTP()
	addr := fmt.Sprintf(":%d", *prism.Port)
	l, e := net.Listen("tcp", addr)
	if e != nil {
		log.Fatalf("Cannot listen on %s: %v", addr, e)
	}
	log.Printf("Prism listening on %s", addr)
	http.Serve(l, nil)
}

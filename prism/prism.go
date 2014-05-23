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

func main() {
	flag.Parse()
	if e := file.Initialize(); e != nil {
		log.Fatalf("file.Initalize() :%v", e)
	}

	s := prism.NewPrism()
	rpc.Register(s)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", *prism.Addr)
	if e != nil {
		log.Fatalf("Cannot listen on %s: %v", *prism.Addr, e)
	}
	log.Printf("Listening on %s", *prism.Addr)
	http.Serve(l, nil)
}

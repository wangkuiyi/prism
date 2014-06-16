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
	"os"
	"os/signal"
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
	go http.Serve(l, nil)

	sig := make(chan os.Signal, 1) // Signal channel must be buffered.
	signal.Notify(sig, os.Interrupt, os.Kill)
	<-sig
	if e := s.KillAll(); e != nil {
		log.Fatal(e)
	}
}

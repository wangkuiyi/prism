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
	"syscall"
)

func main() {
	log.SetPrefix("Prism ")

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

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill, syscall.SIGTERM)
	<-sig
	log.Print("Got signal to kill Prism")
	if e := s.KillAll(); e != nil {
		log.Print("KillAll: ", e)
	}
}

package main

import (
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
)

var (
	addr = flag.String("addr", ":8080", "The listen address")
)

func main() {
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	log.Fatal(http.ListenAndServe(*addr, nil))
}

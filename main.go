package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pnovotnak/daemonless/pkg/manager"

	_ "net/http/pprof"
)

var (
	config = "./daemonless.yaml"
	bind   = "localhost:2000"
	idle   = time.Minute * 10
)

func parseFlags() {
	flag.StringVar(&config, "config", config, "config")
	flag.StringVar(&bind, "bind", bind, "where to bind")
	flag.DurationVar(&idle, "idle", idle, "how long to remain idle before stopping")
	flag.Parse()
}

func main() {
	parseFlags()

	conf, err := manager.LoadConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	router := http.NewServeMux()
	conf.RegisterHTTPHandlers(router)
	server := &http.Server{
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
		IdleTimeout:  500 * time.Millisecond,
		Handler:      router,
		Addr:         bind,
	}

	fmt.Printf("Starting HTTPS server on %s\n", server.Addr)
	log.Fatal(server.ListenAndServe())
}

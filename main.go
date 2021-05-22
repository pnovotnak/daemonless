package main

import (
	"flag"
	"fmt"
	"github.com/pnovotnak/daemonless/pkg/manager"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	bind = "localhost:2000"
)

func parseFlags() {
	flag.Parse()
	flag.StringVar(&bind, "bind", bind, "where to bind")
}

func plexHandlerManager(manager *manager.Manager) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := manager.RunFor(); err != nil {
			log.Println("manager error:", err)
		}
	}
}

type MetaManager struct {
	sync.Mutex

	notify   chan os.Signal
	managers []*manager.Manager
}

func NewMetaManager() *MetaManager {
	return &MetaManager{
		Mutex:    sync.Mutex{},
		notify:   make(chan os.Signal, 2),
		managers: []*manager.Manager{},
	}
}

func (mm *MetaManager) Add(m *manager.Manager) {
	mm.Lock()
	defer mm.Unlock()
	mm.managers = append(mm.managers, m)
}

func (mm *MetaManager) Stop() {
	mm.Lock()
	defer mm.Unlock()
	var err error
	for _, m := range mm.managers {
		if err = m.Stop(); err != nil {
			log.Println("error stopping manager:", err)
		}
	}
}

func (mm *MetaManager) Start() {
	go func() {
		sig := <-mm.notify
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			log.Println("got signal")
			mm.Stop()
			// Without this we never exit
			os.Exit(0)
		}
	}()
	signal.Notify(mm.notify, os.Interrupt, syscall.SIGTERM)
}

func main() {
	parseFlags()

	m := manager.NewManager("/usr/local/bin/plex")

	router := http.NewServeMux()
	router.HandleFunc("/", plexHandlerManager(m))
	server := &http.Server{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  10 * time.Second,
		Handler:      router,
		Addr:         bind,
	}

	// handles signal routing
	mm := NewMetaManager()
	mm.Add(m)
	mm.Start()

	fmt.Printf("Starting HTTPS server on %s\n", server.Addr)
	log.Fatal(server.ListenAndServe())
}

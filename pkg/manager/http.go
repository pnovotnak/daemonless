package manager

// HTTP bindings for managers

import (
	"log"
	"net/http"
)

func (m *Manager) RegisterHTTPHandler(mux *http.ServeMux, path string) {
	url := path + m.URL
	log.Println("registering manager at:", url)
	mux.HandleFunc(url, func(w http.ResponseWriter, r *http.Request) {
		if err := m.Start(); err != nil {
			log.Println("manager error:", err)
		}
	})
}

func (c *Config) RegisterHTTPHandlers(mux *http.ServeMux) {
	for _, m := range c.Managers {
		m.RegisterHTTPHandler(mux, c.RootURL)
	}
}

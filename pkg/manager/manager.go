// +build linux

package manager

import (
	"errors"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	StateStopped = iota
	// Starting state is confined entirely to Manager.Start()
	// StateStarting

	StateRunning
	// StateTerminated is a special state that tells us not to start again
	StateTerminated

	DefaultDuration = time.Minute * 30
)

type Config struct {
	RootURL  string     `yaml:"root_url"`
	Managers []*Manager `yaml:"managers"`
}

func (c *Config) initObjects() {
	for _, m := range c.Managers {
		m.Init()
	}
}

func LoadConfig(path string) (*Config, error) {
	config := &Config{}
	configRaw, err := ioutil.ReadFile(path)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(configRaw, config)
	if err != nil {
		return config, err
	} else if len(config.Managers) == 0 {
		return config, errors.New("malformed configuration")
	}
	config.initObjects()
	return config, nil
}

func (c *Config) RegisterHTTPHandlers(mux *http.ServeMux) {
	for _, m := range c.Managers {
		m.RegisterHTTPHandler(mux, c.RootURL)
	}
}

// Manager is a state machine that manages daemon state
type Manager struct {
	sync.RWMutex

	// Command tell us how to start the daemon
	Command []string `yaml:"command"`
	// Idle configures how long the manager can remain idle before it's child is stopped
	Idle time.Duration `yaml:"idle"`
	// URL is the relative url that will invoke this manager
	URL string `yaml:"url"`

	// TODO can we eliminate State variables by inspecting this value?
	// cmd stores the current execution (if any)
	cmd *exec.Cmd
	// state stores the daemon's current state
	state int
	// timer records when the child should be stopped, stop provides a way to terminate manager goroutines
	timer *time.Timer
	watch chan error
	stop  chan struct{}

	// notify is a channel for signals
	notify chan os.Signal
}

func (m *Manager) signalHandler() {
	go func() {
		var err error
		sig := <-m.notify
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			if err = m.Terminate(); err != nil {
				log.Fatal(err)
			}
			// Without this we never exit
			os.Exit(0)
		}
	}()
	signal.Notify(m.notify, os.Interrupt, syscall.SIGTERM)
}

// expire watches the timer and exits when it expires. Each Start call resets the timer
func (m *Manager) expire() {
	var err error
	select {
	case <-m.timer.C:
		log.Println("timer expired")
		_ = m.Stop()
	case <-m.stop:
		return
	case err = <-m.watch:
		m.Lock()
		m.state = StateStopped
		m.Unlock()
		if err != nil {
			log.Println(err)
		} else {
			log.Println("child exited with code 0")
		}
	}
}

func (m *Manager) Init() {
	m.RWMutex = sync.RWMutex{}
	m.state = 0
	m.timer = time.NewTimer(DefaultDuration)
	// stop is initialized each time the manager is started
	m.notify = make(chan os.Signal, 2)
	m.signalHandler()
}

// Start starts the child process (if not already running)
func (m *Manager) Start() error {
	var err error

	// Within the remainder of this block we are within a Starting state
	m.Lock()
	defer m.Unlock()

	if m.state == StateTerminated {
		// don't start again once terminated
		return nil
	}
	m.timer.Reset(DefaultDuration)
	if m.state == StateRunning {
		// we're already in a good state
		return nil
	}

	// start the manager
	m.stop = make(chan struct{})
	log.Println("starting command")
	m.cmd = exec.Command(m.Command[0], m.Command[1:]...)
	// TODO put these somewhere on disk
	m.cmd.Stdout = os.Stdout
	m.cmd.Stderr = os.Stderr
	m.cmd.SysProcAttr = &syscall.SysProcAttr{
		// Setpgid allows us to correctly target all descendents with signals
		Setpgid: true,
		// Pdeathsig fires a signal upon our death. Note: This doesn't affect internal signal handling> You still need
		// to map (at least) SIGINT and SIGTERM to Terminate()
		Pdeathsig: syscall.SIGTERM,
	}
	if err = m.cmd.Start(); err != nil {
		return err
	}
	m.state = StateRunning
	if m.cmd.Process == nil {
		return errors.New("process failed to start")
	}
	log.Println("will run for:", m.Idle)
	go m.expire()
	go func() {
		m.watch <- m.cmd.Wait()
	}()
	return nil
}

// Stop sends a SIGTERM to the child tree but leaves the manager running
func (m *Manager) Stop() error {
	m.RLock()
	if m.state == StateStopped {
		m.RUnlock()
		return nil
	}
	m.RUnlock()

	m.Lock()
	defer m.Unlock()
	close(m.stop)
	log.Println("stopping child")
	err := syscall.Kill(-m.cmd.Process.Pid, syscall.SIGTERM)
	m.state = StateStopped
	return err
}

// Terminate stops the manager and prevents it from starting again
func (m *Manager) Terminate() error {
	m.Lock()
	m.state = StateTerminated
	m.Unlock()
	return m.Stop()
}
func (m *Manager) RegisterHTTPHandler(mux *http.ServeMux, path string) {
	url := path + m.URL
	log.Println("registering manager at:", url)
	mux.HandleFunc(url, func(w http.ResponseWriter, r *http.Request) {
		if err := m.Start(); err != nil {
			log.Println("manager error:", err)
		}
	})
}

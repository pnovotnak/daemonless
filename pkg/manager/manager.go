// +build linux

package manager

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const (
	StateStopped = iota
	StateStarting
	StateRunning
	// StateTerminated is a special state that tells us not to start again
	StateTerminated

	DefaultDuration = time.Minute * 30
)

// Manager is a state machine that manages daemon state
type Manager struct {
	sync.RWMutex

	// Command tell us how to start the daemon
	Command string `yaml:"command"`

	// cmd stores the current execution (if any)
	cmd *exec.Cmd
	// state stores the daemon's current state
	state int
	// timer records when the child should be stopped, stop provides a way to terminate manager goroutines
	timer *time.Timer
	stop  chan struct{}
}

func NewManager(command string) *Manager {
	return &Manager{
		RWMutex: sync.RWMutex{},
		Command: command,
		state:   0,
		timer:   time.NewTimer(DefaultDuration),
		// stop is initialized each time the manager is started
	}
}

func (m *Manager) watch() {
	// TODO possible thread safety issue
	err := m.cmd.Wait()
	if err != nil {
		log.Println(err)
	} else {
		log.Println("command exited with status 0")
	}
	m.Lock()
	m.state = StateStopped
	m.Unlock()
}

func (m *Manager) expire() {
	select {
	case <-m.timer.C:
		log.Println("timer expired")
		_ = m.Stop()
	case <-m.stop:
		return
	}
}

// RunFor runs the manager (if not already running)
func (m *Manager) RunFor() error {
	var err error

	m.RLock()
	if m.state == StateTerminated {
		return nil
	}
	m.timer.Reset(DefaultDuration)
	if m.state == StateRunning || m.state == StateStarting {
		// we're already in a good state
		m.RUnlock()
		return nil
	}
	m.RUnlock()

	// start the manager
	m.Lock()
	defer m.Unlock()
	m.stop = make(chan struct{})
	m.state = StateStarting
	log.Println("starting command")
	c := exec.Command(m.Command)
	// TODO put these somewhere on disk
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.SysProcAttr = &syscall.SysProcAttr{
		// Setpgid allows us to correctly target all descendents with signals
		Setpgid: true,
		// Pdeathsig fires a signal upon our death
		Pdeathsig: syscall.SIGTERM,
	}
	if err = c.Start(); err != nil {
		return err
	}
	m.state = StateRunning
	if c.Process == nil {
		return errors.New("process failed to start")
	}
	log.Println("will run for:", DefaultDuration)
	m.cmd = c
	go m.watch()
	go m.expire()
	return nil
}

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

package manager

import (
	"syscall"
	"testing"
	"time"

	_ "net/http/pprof"
)

const (
	configPath = "../../example/hello-world/daemonless.yaml"
)

func getManager(t *testing.T, config string) *Manager {
	var (
		err error
		c   *Config
	)

	if c, err = LoadConfig(config); err != nil {
		t.Fatal(err)
	}
	return c.Managers[0]
}

func TestManager_Start(t *testing.T) {
	var err error
	t.Parallel()

	m := getManager(t, configPath)
	if err = m.Start(); err != nil {
		t.Fatal(err)
	}
	// The manager should continue to run for 0.5s
	time.Sleep(250 * time.Millisecond)
	if m.GetStatus() != StateRunning {
		t.Fatal("bad state:", m.GetStatus())
	}
	_ = m.Start()
	time.Sleep(250 * time.Millisecond)
	if m.GetStatus() != StateRunning {
		t.Fatal("bad state:", m.GetStatus())
	}
	// The manager should stop at around 0.5s, but we allow 100ms of jitter
	time.Sleep(350 * time.Millisecond)
	if m.GetStatus() != StateStopped {
		t.Fatal("bad state:", m.GetStatus())
	}
}

func TestManager_Terminate(t *testing.T) {
	var err error
	t.Parallel()

	m := getManager(t, configPath)
	if err = m.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(250 * time.Millisecond)
	err = m.cmd.Process.Signal(syscall.Signal(0))
	if err != nil {
		t.Fatal("unable to find child process")
	}
	if err = m.Terminate(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	err = m.cmd.Process.Signal(syscall.Signal(0))
	if err == nil {
		t.Fatal("child process still running")
	}
	status := m.GetStatus()
	if status != StateTerminated {
		t.Fatal()
	}
}

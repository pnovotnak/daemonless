package manager

import (
	"testing"
	"time"
)

func TestManagerLifecycle(t *testing.T) {
	var (
		err error
		c   *Config
	)

	c, err = LoadConfig("examples/hello-world")
	if err != nil {
		t.Fatal(err)
	}
	err = c.Managers[0].Start()
	time.Sleep(time.Second)

}

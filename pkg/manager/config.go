package manager

import (
	"errors"
	"io/ioutil"

	"gopkg.in/yaml.v3"
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

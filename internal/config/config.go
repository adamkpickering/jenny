package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const configPath = "configuration.yaml"

type ConfigYaml struct {
	Input     string `yaml:"Input"`
	Output    string `yaml:"Output"`
	Templates string `yaml:"Templates"`
}

func Get() (ConfigYaml, error) {
	configYaml := ConfigYaml{}

	fd, err := os.Open(configPath)
	if !errors.Is(err, os.ErrNotExist) {
		if err != nil {
			return ConfigYaml{}, fmt.Errorf("failed to open: %s", err)
		}
		defer fd.Close()
		decoder := yaml.NewDecoder(fd)
		if err := decoder.Decode(&configYaml); err != nil {
			return ConfigYaml{}, fmt.Errorf("failed to parse: %s", err)
		}
	}

	configYaml.setDefaults()

	return configYaml, nil
}

func (configYaml *ConfigYaml) setDefaults() {
	if configYaml.Input == "" {
		configYaml.Input = "input"
	}
	if configYaml.Output == "" {
		configYaml.Output = "output"
	}
	if configYaml.Templates == "" {
		configYaml.Templates = "templates"
	}
}

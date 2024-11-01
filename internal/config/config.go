package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ConfigYaml struct {
	Content   string `json:"content"`
	Output    string `json:"output"`
	Templates string `json:"templates"`
}

func ReadFile(configYamlPath string) (ConfigYaml, error) {
	configYaml := ConfigYaml{}

	fd, err := os.Open(configYamlPath)
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
	if configYaml.Content == "" {
		configYaml.Content = "content"
	}
	if configYaml.Output == "" {
		configYaml.Output = "output"
	}
	if configYaml.Templates == "" {
		configYaml.Templates = "templates"
	}
}

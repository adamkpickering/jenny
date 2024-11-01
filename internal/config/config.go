package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type ConfigJson struct {
	Content   string `json:"content"`
	Output    string `json:"output"`
	Templates string `json:"templates"`
}

func Read(configJsonPath string) (ConfigJson, error) {
	configJson := ConfigJson{}

	fd, err := os.Open(configJsonPath)
	if !errors.Is(err, os.ErrNotExist) {
		if err != nil {
			return ConfigJson{}, fmt.Errorf("failed to open: %s", err)
		}
		defer fd.Close()
		decoder := json.NewDecoder(fd)
		if err := decoder.Decode(&configJson); err != nil {
			return ConfigJson{}, fmt.Errorf("failed to parse: %s", err)
		}
	}

	configJson.setDefaults()

	return configJson, nil
}

func (configJson *ConfigJson) setDefaults() {
	if configJson.Content == "" {
		configJson.Content = "content"
	}
	if configJson.Output == "" {
		configJson.Output = "output"
	}
	if configJson.Templates == "" {
		configJson.Templates = "templates"
	}
}
